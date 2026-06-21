package collector

func scoreModules(
	modules []ModuleCandidate,
	subtreeFiles []int,
	parents []int,
	totalFiles int,
) ([]int, []float64, []float64) {
	scores := make([]int, len(modules))
	extensionOverlap := make([]float64, len(modules))
	categoryOverlap := make([]float64, len(modules))

	for i, module := range modules {
		score := compressionScore(module, subtreeFiles[i], totalFiles)

		if parents[i] != -1 {
			parent := modules[parents[i]]
			extensionOverlap[i] = sliceOverlap(module.DominantExtensions, parent.DominantExtensions)
			categoryOverlap[i] = mapKeyOverlap(module.EvidenceByCategory, parent.EvidenceByCategory)
		}
		if parents[i] != -1 && isHighlySimilarToParent(extensionOverlap[i], categoryOverlap[i]) {
			score -= redundantChildPenalty
		}
		scores[i] = score
	}
	return scores, extensionOverlap, categoryOverlap
}

func compressionScore(module ModuleCandidate, subtreeFiles, totalFiles int) int {
	return module.EvidenceCount*compressionEvidenceWeight +
		module.FileCount +
		repositoryCoveragePercent(subtreeFiles, totalFiles)*coverageScoreWeight
}

func repositoryCoveragePercent(subtreeFiles, totalFiles int) int {
	if totalFiles == 0 {
		return 0
	}
	return subtreeFiles * coveragePercentScale / totalFiles
}

func isHighlySimilarToParent(extensionOverlap, categoryOverlap float64) bool {
	return extensionOverlap >= highOverlapThreshold &&
		categoryOverlap >= highOverlapThreshold
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
