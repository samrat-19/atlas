package collector

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"atlas/internal/model"
)

// Collect walks the directory tree rooted at root and returns its structural
// inventory and inferred module hierarchy.
func Collect(root string) (model.Result, error) {
	var res model.Result

	// profile is the single source of every tunable threshold and weight
	// used below. Only model.DefaultHeuristics exists today; threading it
	// explicitly, rather than reading package constants deep inside scoring
	// and compression, is what makes a future alternate profile possible
	// without changing those functions.
	profile := model.DefaultHeuristics

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return res, err
	}
	res.Root = absRoot

	var evidence []model.EvidenceItem
	var pruned []model.PrunedPath
	totalFiles := 0
	topology := newTopologySummary()
	clusters := newClusterSummary()
	census := newCensusSummary()
	extensions := newExtensionSummary()
	dirStats := make(map[string]*dirStat)

	walkFn := func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(absRoot, absPath)
		if err != nil {
			return err
		}

		// Skip well-known operational directories that are never source.
		// We check relPath != "." so we never accidentally prune the scan
		// root itself (WalkDir visits the root first with relPath == ".").
		if entry.IsDir() && relPath != "." {
			if policy, ok := prunePolicy(entry.Name()); ok {
				pruned = append(pruned, model.PrunedPath{
					RelativePath: filepath.ToSlash(relPath),
					Policy:       policy,
				})
				return fs.SkipDir
			}
		}

		topDir := topLevelDirectoryForRelativePath(relPath)
		if !entry.IsDir() {
			totalFiles++
			updateCensusWithFile(&census, topDir)

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			updateExtensionsWithFile(&extensions, topDir, ext)
			directoryStatsFor(dirStats, relPath).updateWithFile(ext)
		}

		item, ok := createEvidenceItem(entry, absPath, relPath, profile)
		if !ok {
			return nil
		}

		evidence = append(evidence, item)
		updateTopologyWithEvidence(&topology, item)
		updateClustersWithEvidence(&clusters, item)
		updateCensusWithEvidence(&census, topDir)
		directoryStatsFor(dirStats, relPath).updateWithEvidence(item)
		return nil
	}

	if err := filepath.WalkDir(absRoot, walkFn); err != nil {
		return res, err
	}

	census.TotalDirectories = len(census.Directories)
	moduleSummary := buildModuleSummary(dirStats, profile)

	// Role classification is independent of scoring — see classifyRole in
	// role.go. Done here, after buildModuleSummary returns, rather than
	// inside it, so the scoring package never imports or calls into
	// classification at all; orchestration is the only thing that ties the
	// two together.
	for i := range moduleSummary.Modules {
		m := &moduleSummary.Modules[i]
		m.Role = classifyRole(m.Path, m.EvidenceCount, m.EvidenceStrength, profile)
	}

	compressedModules := compressModules(moduleSummary.Modules, dirStats, totalFiles, profile)
	unrecognized := buildUnrecognizedSummary(moduleSummary.Modules, profile)

	// Sort pruned paths by relative path so the Result is byte-identical
	// across repeated scans. WalkDir already walks lexically, but making
	// the order an explicit contract here means it stays stable even if the
	// traversal strategy changes later.
	sort.Slice(pruned, func(i, j int) bool {
		return pruned[i].RelativePath < pruned[j].RelativePath
	})

	res.SchemaVersion = model.CurrentSchemaVersion
	res.TotalFiles = totalFiles
	res.PrunedPaths = pruned
	res.Evidence = evidence
	res.TopologySummary = topology
	res.ClusterSummary = clusters
	res.CensusSummary = census
	res.ExtensionSummary = extensions
	res.ModuleSummary = moduleSummary
	res.CompressedModuleSummary = compressedModules
	res.HierarchySummary = buildHierarchy(compressedModules.Modules)
	res.UnrecognizedSummary = unrecognized
	return res, nil
}
