package collector

import (
	"io/fs"
	"path/filepath"

	"atlas/internal/model"
)

func newTopologySummary() model.TopologySummary {
	return model.TopologySummary{
		EvidenceCountByCategory: make(map[string]int),
		EvidenceCountByFilename: make(map[string]int),
	}
}

func newClusterSummary() model.ClusterSummary {
	return model.ClusterSummary{Clusters: make(map[string]model.EvidenceCluster)}
}

func newCensusSummary() model.CensusSummary {
	return model.CensusSummary{Directories: make(map[string]model.DirectoryCensus)}
}

func newExtensionSummary() model.ExtensionSummary {
	return model.ExtensionSummary{
		ByExtension: make(map[string]int),
		ByCluster:   make(map[string]map[string]int),
	}
}

func createEvidenceItem(entry fs.DirEntry, absPath, relPath string, profile model.HeuristicProfile) (model.EvidenceItem, bool) {
	key, category, confidence, ok := MatchEvidence(entry.Name(), relPath, profile)
	if !ok {
		return model.EvidenceItem{}, false
	}
	return model.EvidenceItem{
		Filename:     key,
		AbsolutePath: absPath,
		RelativePath: relPath,
		Category:     category,
		Confidence:   confidence,
	}, true
}

func updateTopologyWithEvidence(s *model.TopologySummary, item model.EvidenceItem) {
	s.TotalEvidenceItems++
	s.EvidenceCountByCategory[item.Category]++
	s.EvidenceCountByFilename[item.Filename]++

	if filepath.Dir(item.RelativePath) == "." {
		s.RootEvidenceCount++
	} else {
		s.NestedEvidenceCount++
	}
}

func updateClustersWithEvidence(s *model.ClusterSummary, item model.EvidenceItem) {
	clusterName := topLevelDirectoryForRelativePath(item.RelativePath)
	cluster, ok := s.Clusters[clusterName]
	if !ok {
		cluster = model.EvidenceCluster{
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

func updateCensusWithFile(s *model.CensusSummary, directory string) {
	entry := s.Directories[directory]
	entry.DirectoryName = directory
	entry.TotalFiles++
	s.Directories[directory] = entry
}

func updateCensusWithEvidence(s *model.CensusSummary, directory string) {
	entry := s.Directories[directory]
	entry.DirectoryName = directory
	entry.EvidenceItemCount++
	s.Directories[directory] = entry
}

func updateExtensionsWithFile(s *model.ExtensionSummary, cluster, extension string) {
	s.ByExtension[extension]++
	if _, ok := s.ByCluster[cluster]; !ok {
		s.ByCluster[cluster] = make(map[string]int)
	}
	s.ByCluster[cluster][extension]++
}
