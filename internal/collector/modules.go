package collector

import "sort"

func buildModuleSummary(dirStats map[string]*dirStat) ModuleSummary {
	var modules []ModuleCandidate
	for path, stats := range dirStats {
		if !isModuleCandidate(stats) {
			continue
		}
		modules = append(modules, newModuleCandidate(path, stats))
	}

	sort.Slice(modules, func(i, j int) bool {
		return moduleCandidateScore(modules[i]) > moduleCandidateScore(modules[j])
	})
	return ModuleSummary{TotalModules: len(modules), Modules: modules}
}

func isModuleCandidate(stats *dirStat) bool {
	density := 0.0
	if stats.FileCount > 0 {
		density = float64(stats.EvidenceCount) / float64(stats.FileCount)
	}
	return stats.EvidenceCount > 0 ||
		stats.FileCount >= 200 ||
		(stats.FileCount >= 20 && density >= 0.05)
}

func newModuleCandidate(path string, stats *dirStat) ModuleCandidate {
	return ModuleCandidate{
		Path:               path,
		FileCount:          stats.FileCount,
		EvidenceCount:      stats.EvidenceCount,
		DominantExtensions: dominantExtensions(stats.Extensions, 3),
		EvidenceByCategory: copyCountMap(stats.EvidenceByCategory),
		EvidenceByFilename: copyCountMap(stats.EvidenceByFilename),
	}
}

func dominantExtensions(extensions map[string]int, limit int) []string {
	type extensionCount struct {
		extension string
		count     int
	}

	counts := make([]extensionCount, 0, len(extensions))
	for extension, count := range extensions {
		counts = append(counts, extensionCount{extension: extension, count: count})
	}
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	result := make([]string, 0, limit)
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, counts[i].extension)
	}
	return result
}

func moduleCandidateScore(module ModuleCandidate) int {
	return module.EvidenceCount*100 + module.FileCount
}

func copyCountMap(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
