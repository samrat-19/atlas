package collector

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Collect walks the directory tree rooted at `root` and counts files and
// discovers evidence items as defined in the EvidenceRegistry. It performs a
// single filesystem walk and returns the aggregated Result.
func Collect(root string) (Result, error) {
	var res Result

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return res, err
	}
	res.Root = absRoot

	count := 0
	var evidence []EvidenceItem
	summary := TopologySummary{
		EvidenceCountByCategory: make(map[string]int),
		EvidenceCountByFilename: make(map[string]int),
	}
	clusterSummary := ClusterSummary{
		Clusters: make(map[string]EvidenceCluster),
	}
	censusSummary := CensusSummary{
		Directories: make(map[string]DirectoryCensus),
	}
	extensionSummary := ExtensionSummary{
		ByExtension: make(map[string]int),
		ByCluster:   make(map[string]map[string]int),
	}

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			count++
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
		if !d.IsDir() {
			censusSummary.updateWithFile(topDir)
			// collect extension stats
			ext := strings.ToLower(filepath.Ext(d.Name()))
			// filepath.Ext returns "" for files without extensions; we keep that
			extensionSummary.ByExtension[ext]++
			// ensure cluster map exists
			if _, ok := extensionSummary.ByCluster[topDir]; !ok {
				extensionSummary.ByCluster[topDir] = make(map[string]int)
			}
			extensionSummary.ByCluster[topDir][ext]++
		}

		if item, ok := createEvidenceItem(d, absPath, relPath); ok {
			evidence = append(evidence, item)
			summary.updateWithEvidence(item)
			clusterSummary.updateWithEvidence(item)
			censusSummary.updateWithEvidence(topDir)
		}
		return nil
	}

	if err := filepath.WalkDir(absRoot, walkFn); err != nil {
		return res, err
	}

	censusSummary.TotalDirectories = len(censusSummary.Directories)
	res.TotalFiles = count
	res.Evidence = evidence
	res.TopologySummary = summary
	res.ClusterSummary = clusterSummary
	res.CensusSummary = censusSummary
	res.ExtensionSummary = extensionSummary
	return res, nil
}

func createEvidenceItem(d fs.DirEntry, absPath, relPath string) (EvidenceItem, bool) {
	// Use the registry matcher which handles normalization and indexing.
	if key, category, ok := MatchEvidence(d.Name(), relPath); ok {
		return EvidenceItem{Filename: key, AbsolutePath: absPath, RelativePath: relPath, Category: category}, true
	}
	return EvidenceItem{}, false
}

func (s *TopologySummary) updateWithEvidence(item EvidenceItem) {
	s.TotalEvidenceItems++
	s.EvidenceCountByCategory[item.Category]++
	s.EvidenceCountByFilename[item.Filename]++

	if filepath.Dir(item.RelativePath) == "." {
		s.RootEvidenceCount++
	} else {
		s.NestedEvidenceCount++
	}
}

func (s *ClusterSummary) updateWithEvidence(item EvidenceItem) {
	clusterName := topLevelDirectoryForRelativePath(item.RelativePath)
	cluster, ok := s.Clusters[clusterName]
	if !ok {
		cluster = EvidenceCluster{
			ClusterName:             clusterName,
			EvidenceCountByCategory: make(map[string]int),
			EvidenceCountByFilename: make(map[string]int),
		}
	}

	cluster.EvidenceItemCount++
	cluster.EvidenceCountByCategory[item.Category]++
	cluster.EvidenceCountByFilename[item.Filename]++

	s.Clusters[clusterName] = cluster
}

func (s *CensusSummary) updateWithFile(directory string) {
	entry, ok := s.Directories[directory]
	if !ok {
		entry = DirectoryCensus{DirectoryName: directory}
	}
	entry.TotalFiles++
	s.Directories[directory] = entry
}

func (s *CensusSummary) updateWithEvidence(directory string) {
	entry, ok := s.Directories[directory]
	if !ok {
		entry = DirectoryCensus{DirectoryName: directory}
	}
	entry.EvidenceItemCount++
	s.Directories[directory] = entry
}

func topLevelDirectoryForRelativePath(relPath string) string {
	normRelPath := filepath.ToSlash(relPath)
	parts := strings.Split(normRelPath, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "_root"
	}
	if len(parts) == 1 {
		return "_root"
	}
	return parts[0]
}
