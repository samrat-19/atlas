package collector

import (
	"sort"

	"atlas/internal/model"
)

// buildUnrecognizedSummary finds module candidates that qualified purely by
// size — no evidence at all (see isLargeDirectory) — and groups them by
// shared dominant extension. It does not change scoring, compression, or
// retention; it only looks at the same candidate list those steps already
// produced, from a different angle: "what large, unexplained directories
// keep recurring, and what do they have in common?"
//
// modules should be the full pre-compression candidate list
// (ModuleSummary.Modules), not the retained/compressed list — the point is
// to find registry gaps across every directory Atlas considered, including
// ones compression later dropped as redundant.
func buildUnrecognizedSummary(modules []model.ModuleCandidate, profile model.HeuristicProfile) model.UnrecognizedSummary {
	type clusterAccumulator struct {
		directoryCount int
		totalFiles     int
		examplePaths   []string
	}
	accumulators := make(map[string]*clusterAccumulator)

	summary := model.UnrecognizedSummary{}
	for _, module := range modules {
		if module.EvidenceCount != 0 {
			continue // Atlas has at least some explanation for this one.
		}
		summary.TotalUnrecognizedDirectories++
		summary.TotalUnrecognizedFiles += module.FileCount

		for _, extension := range module.DominantExtensions {
			acc, ok := accumulators[extension]
			if !ok {
				acc = &clusterAccumulator{}
				accumulators[extension] = acc
			}
			acc.directoryCount++
			acc.totalFiles += module.FileCount
			if len(acc.examplePaths) < profile.Diagnostics.ExampleDirectoryLimit {
				acc.examplePaths = append(acc.examplePaths, module.Path)
			}
		}
	}

	clusters := make([]model.UnrecognizedExtensionCluster, 0, len(accumulators))
	for extension, acc := range accumulators {
		examplePaths := append([]string(nil), acc.examplePaths...)
		sort.Strings(examplePaths)
		clusters = append(clusters, model.UnrecognizedExtensionCluster{
			Extension:      extension,
			DirectoryCount: acc.directoryCount,
			TotalFiles:     acc.totalFiles,
			ExamplePaths:   examplePaths,
		})
	}
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].DirectoryCount != clusters[j].DirectoryCount {
			return clusters[i].DirectoryCount > clusters[j].DirectoryCount
		}
		if clusters[i].TotalFiles != clusters[j].TotalFiles {
			return clusters[i].TotalFiles > clusters[j].TotalFiles
		}
		return clusters[i].Extension < clusters[j].Extension
	})

	summary.Clusters = clusters
	return summary
}
