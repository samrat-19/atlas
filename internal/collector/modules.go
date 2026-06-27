package collector

import "sort"

// buildModuleSummary turns per-directory stats into the candidate module
// list: which folders qualify (isModuleCandidate), how they're scored
// (moduleCandidateScore), sorted strongest first. profile supplies every
// threshold and weight used along the way — see HeuristicProfile in
// heuristics.go for what each field means and why.
func buildModuleSummary(dirStats map[string]*dirStat, profile HeuristicProfile) ModuleSummary {
	var modules []ModuleCandidate
	for path, stats := range dirStats {
		if !isModuleCandidate(stats, profile) {
			continue
		}
		modules = append(modules, newModuleCandidate(path, stats, profile))
	}

	sort.Slice(modules, func(i, j int) bool {
		scoreI, scoreJ := moduleCandidateScore(modules[i], profile), moduleCandidateScore(modules[j], profile)
		if scoreI == scoreJ {
			return modules[i].Path < modules[j].Path
		}
		return scoreI > scoreJ
	})
	return ModuleSummary{TotalModules: len(modules), Modules: modules}
}

// isModuleCandidate answers candidate-selection Rule 1–3 from
// docs/heuristics.md: a folder qualifies if it has any evidence, is large
// enough to matter on its own, or is dense enough with evidence relative to
// its size.
func isModuleCandidate(stats *dirStat, profile HeuristicProfile) bool {
	return hasEvidence(stats) ||
		isLargeDirectory(stats, profile) ||
		hasHighEvidenceDensity(stats, profile)
}

func hasEvidence(stats *dirStat) bool {
	return stats.EvidenceCount > 0
}

func isLargeDirectory(stats *dirStat, profile HeuristicProfile) bool {
	return stats.FileCount >= profile.CandidateSelection.LargeDirectoryFileThreshold
}

func hasHighEvidenceDensity(stats *dirStat, profile HeuristicProfile) bool {
	if stats.FileCount < profile.CandidateSelection.EvidenceDenseFileThreshold {
		return false
	}
	return evidenceDensity(stats) >= profile.CandidateSelection.EvidenceDensityThreshold
}

func evidenceDensity(stats *dirStat) float64 {
	density := 0.0
	if stats.FileCount > 0 {
		density = float64(stats.EvidenceCount) / float64(stats.FileCount)
	}
	return density
}

func newModuleCandidate(path string, stats *dirStat, profile HeuristicProfile) ModuleCandidate {
	strength := evidenceStrength(stats)
	return ModuleCandidate{
		Path:               path,
		FileCount:          stats.FileCount,
		EvidenceCount:      stats.EvidenceCount,
		DominantExtensions: dominantExtensions(stats.Extensions, profile.Scoring.DominantExtensionLimit),
		EvidenceByCategory: copyCountMap(stats.EvidenceByCategory),
		EvidenceByFilename: copyCountMap(stats.EvidenceByFilename),
		EvidenceStrength:   strength,
		NoiseProbability:   noiseProbability(stats.EvidenceCount, strength),
	}
}

// evidenceStrength is a directory's average evidence confidence: how strong
// its own evidence matches are, independent of how many there are or how it
// compares to any parent. A directory with no evidence has no confidence
// signal to average, so it is 0 — not "weak evidence," just "no evidence."
func evidenceStrength(stats *dirStat) float64 {
	if stats.EvidenceCount == 0 {
		return 0
	}
	return stats.EvidenceConfidenceSum / float64(stats.EvidenceCount)
}

// noiseProbability estimates how likely a candidate is incidental noise
// rather than a real boundary, from the one signal Atlas actually has:
// evidence confidence. A candidate with no evidence at all (qualified purely
// by size — see isLargeDirectory) gets a neutral 0.5, not a high noise
// estimate: docs/heuristics.md's own Rule 2 is that a large directory with no
// build file of its own can still be genuinely important, so asserting it is
// probably noise would contradict the reason that rule exists.
func noiseProbability(evidenceCount int, strength float64) float64 {
	if evidenceCount == 0 {
		return 0.5
	}
	return 1 - strength
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
		if counts[i].count == counts[j].count {
			return counts[i].extension < counts[j].extension
		}
		return counts[i].count > counts[j].count
	})

	result := make([]string, 0, limit)
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, counts[i].extension)
	}
	return result
}

func moduleCandidateScore(module ModuleCandidate, profile HeuristicProfile) int {
	return module.EvidenceCount*profile.Scoring.ModuleEvidenceWeight + module.FileCount
}

func copyCountMap(source map[string]int) map[string]int {
	result := make(map[string]int, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
