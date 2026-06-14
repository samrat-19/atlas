package collector

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Collect walks the directory tree rooted at root and returns its structural
// inventory and inferred module hierarchy.
func Collect(root string) (Result, error) {
	var res Result

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return res, err
	}
	res.Root = absRoot

	var evidence []EvidenceItem
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

		topDir := topLevelDirectoryForRelativePath(relPath)
		if !entry.IsDir() {
			totalFiles++
			census.updateWithFile(topDir)

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			extensions.updateWithFile(topDir, ext)
			directoryStatsFor(dirStats, relPath).updateWithFile(ext)
		}

		item, ok := createEvidenceItem(entry, absPath, relPath)
		if !ok {
			return nil
		}

		evidence = append(evidence, item)
		topology.updateWithEvidence(item)
		clusters.updateWithEvidence(item)
		census.updateWithEvidence(topDir)
		directoryStatsFor(dirStats, relPath).updateWithEvidence(item)
		return nil
	}

	if err := filepath.WalkDir(absRoot, walkFn); err != nil {
		return res, err
	}

	census.TotalDirectories = len(census.Directories)
	moduleSummary := buildModuleSummary(dirStats)
	compressedModules := compressModules(moduleSummary.Modules, dirStats, totalFiles)

	res.TotalFiles = totalFiles
	res.Evidence = evidence
	res.TopologySummary = topology
	res.ClusterSummary = clusters
	res.CensusSummary = census
	res.ExtensionSummary = extensions
	res.ModuleSummary = moduleSummary
	res.CompressedModuleSummary = compressedModules
	res.HierarchySummary = buildHierarchy(compressedModules.Modules)
	return res, nil
}
