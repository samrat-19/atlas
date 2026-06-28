package collector

import "atlas/internal/model"

// moduleScore holds every per-candidate result scoreModules computes during
// compression: the legacy compression score plus the explainable dimensions
// introduced in Phase 2 D3. ExtensionOverlap and CategoryOverlap are raw
// overlap ratios (not yet folded into NoveltyVsParent's "best of either
// axis" combination) — retainedModules needs them separately because its
// retention rule treats "novel by extension" and "novel by category" as
// independent ways to survive, not a single blended number.
type moduleScore struct {
	Score                int
	ExtensionOverlap     float64
	CategoryOverlap      float64
	StructuralProminence float64
	NoveltyVsParent      float64
	BoundaryConfidence   float64
}

// scoreModules computes the compression-stage score for every candidate,
// plus how similar each is to its parent (by extension mix and evidence
// category), which compressModules uses to decide what survives. profile
// supplies every weight and threshold involved — see model.CompressionConfig.
func scoreModules(
	modules []model.ModuleCandidate,
	subtreeFiles []int,
	parents []int,
	totalFiles int,
	profile model.HeuristicProfile,
) []moduleScore {
	results := make([]moduleScore, len(modules))

	for i, module := range modules {
		score := compressionScore(module, subtreeFiles[i], totalFiles, profile)
		structuralProminence := float64(repositoryCoveragePercent(subtreeFiles[i], totalFiles, profile)) / 100.0

		// novelty defaults to fully novel: a candidate with no parent has
		// nothing to be redundant with.
		var extensionOverlap, categoryOverlap float64
		noveltyVsParent := 1.0

		if parents[i] != -1 {
			parent := modules[parents[i]]
			extensionOverlap = sliceOverlap(module.DominantExtensions, parent.DominantExtensions)
			categoryOverlap = mapKeyOverlap(module.EvidenceByCategory, parent.EvidenceByCategory)
			noveltyVsParent = noveltyFromOverlap(extensionOverlap, categoryOverlap)

			if isHighlySimilarToParent(extensionOverlap, categoryOverlap, profile) {
				score -= profile.Compression.RedundantChildPenalty
			}
		}

		results[i] = moduleScore{
			Score:                score,
			ExtensionOverlap:     extensionOverlap,
			CategoryOverlap:      categoryOverlap,
			StructuralProminence: structuralProminence,
			NoveltyVsParent:      noveltyVsParent,
			// Strong, confident evidence and/or a meaningful difference
			// from the parent both support treating a candidate as a real
			// boundary rather than redundant noise — see ModuleCandidate's
			// doc comment in internal/model/types.go for the full
			// rationale.
			BoundaryConfidence: (module.EvidenceStrength + noveltyVsParent) / 2,
		}
	}
	return results
}

// noveltyFromOverlap combines extension-mix novelty and evidence-category
// novelty into the single NoveltyVsParent dimension by taking whichever axis
// shows more difference. This mirrors isNovelComparedToParent's existing OR
// semantics in retainedModules: being different by either axis is enough to
// be considered novel, so the combined score should reflect the strongest
// of the two signals, not their average.
func noveltyFromOverlap(extensionOverlap, categoryOverlap float64) float64 {
	extensionNovelty := 1.0 - extensionOverlap
	categoryNovelty := 1.0 - categoryOverlap
	if extensionNovelty > categoryNovelty {
		return extensionNovelty
	}
	return categoryNovelty
}

func compressionScore(module model.ModuleCandidate, subtreeFiles, totalFiles int, profile model.HeuristicProfile) int {
	return module.EvidenceCount*profile.Compression.CompressionEvidenceWeight +
		module.FileCount +
		repositoryCoveragePercent(subtreeFiles, totalFiles, profile)*profile.Compression.CoverageScoreWeight
}

func repositoryCoveragePercent(subtreeFiles, totalFiles int, profile model.HeuristicProfile) int {
	if totalFiles == 0 {
		return 0
	}
	return subtreeFiles * profile.Compression.CoveragePercentScale / totalFiles
}

func isHighlySimilarToParent(extensionOverlap, categoryOverlap float64, profile model.HeuristicProfile) bool {
	return extensionOverlap >= profile.Compression.HighOverlapThreshold &&
		categoryOverlap >= profile.Compression.HighOverlapThreshold
}

func sliceOverlap(left, right []string) float64 {
	leftSet := make(map[string]struct{}, len(left))
	for _, value := range left {
		leftSet[value] = struct{}{}
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}

	intersection := 0
	for value := range leftSet {
		if _, ok := rightSet[value]; ok {
			intersection++
		}
	}
	union := len(leftSet)
	if len(rightSet) > union {
		union = len(rightSet)
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func mapKeyOverlap(left, right map[string]int) float64 {
	intersection := 0
	for key := range left {
		if _, ok := right[key]; ok {
			intersection++
		}
	}
	union := len(left)
	if len(right) > union {
		union = len(right)
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
