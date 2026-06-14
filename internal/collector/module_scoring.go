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
		coverage := 0
		if totalFiles > 0 {
			coverage = subtreeFiles[i] * 100 / totalFiles
		}
		score := module.EvidenceCount*200 + module.FileCount + coverage*10

		if parents[i] != -1 {
			parent := modules[parents[i]]
			extensionOverlap[i] = sliceOverlap(module.DominantExtensions, parent.DominantExtensions)
			categoryOverlap[i] = mapKeyOverlap(module.EvidenceByCategory, parent.EvidenceByCategory)
		}
		if parents[i] != -1 && extensionOverlap[i] >= 0.9 && categoryOverlap[i] >= 0.9 {
			score -= 500
		}
		scores[i] = score
	}
	return scores, extensionOverlap, categoryOverlap
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
