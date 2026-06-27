package collector

import (
	"io/fs"
	"path/filepath"
)

func newTopologySummary() TopologySummary {
	return TopologySummary{
		EvidenceCountByCategory: make(map[string]int),
		EvidenceCountByFilename: make(map[string]int),
	}
}

func newClusterSummary() ClusterSummary {
	return ClusterSummary{Clusters: make(map[string]EvidenceCluster)}
}

func newCensusSummary() CensusSummary {
	return CensusSummary{Directories: make(map[string]DirectoryCensus)}
}

func newExtensionSummary() ExtensionSummary {
	return ExtensionSummary{
		ByExtension: make(map[string]int),
		ByCluster:   make(map[string]map[string]int),
	}
}

func createEvidenceItem(entry fs.DirEntry, absPath, relPath string, profile HeuristicProfile) (EvidenceItem, bool) {
	key, category, confidence, ok := MatchEvidence(entry.Name(), relPath, profile)
	if !ok {
		return EvidenceItem{}, false
	}
	return EvidenceItem{
		Filename:     key,
		AbsolutePath: absPath,
		RelativePath: relPath,
		Category:     category,
		Confidence:   confidence,
	}, true
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
	entry := s.Directories[directory]
	entry.DirectoryName = directory
	entry.TotalFiles++
	s.Directories[directory] = entry
}

func (s *CensusSummary) updateWithEvidence(directory string) {
	entry := s.Directories[directory]
	entry.DirectoryName = directory
	entry.EvidenceItemCount++
	s.Directories[directory] = entry
}

func (s *ExtensionSummary) updateWithFile(cluster, extension string) {
	s.ByExtension[extension]++
	if _, ok := s.ByCluster[cluster]; !ok {
		s.ByCluster[cluster] = make(map[string]int)
	}
	s.ByCluster[cluster][extension]++
}
