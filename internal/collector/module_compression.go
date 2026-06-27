package collector

import (
	"path/filepath"
	"sort"
	"strings"
)

// compressModules prunes redundant parent-child candidates, keeping a child
// only when it scores strongly enough on its own or looks different enough
// from its parent (see retainedModules). profile supplies every weight and
// threshold scoreModules and retainedModules read — see CompressionConfig in
// heuristics.go.
func compressModules(
	modules []ModuleCandidate,
	dirStats map[string]*dirStat,
	totalFiles int,
	profile HeuristicProfile,
) CompressedModuleSummary {
	summary := CompressedModuleSummary{TotalCandidates: len(modules)}
	if len(modules) == 0 {
		return summary
	}

	normalizedPaths, index := indexModulePaths(modules)
	subtreeFiles := moduleSubtreeFileCounts(normalizedPaths, dirStats)
	parents := moduleParents(normalizedPaths, index)
	scores := scoreModules(modules, subtreeFiles, parents, totalFiles, profile)
	retained := retainedModules(normalizedPaths, parents, scores, profile)

	for i, keep := range retained {
		if !keep {
			continue
		}
		module := modules[i]
		module.Score = scores[i].Score
		module.StructuralProminence = scores[i].StructuralProminence
		module.NoveltyVsParent = scores[i].NoveltyVsParent
		module.BoundaryConfidence = scores[i].BoundaryConfidence
		summary.Modules = append(summary.Modules, module)
	}

	summary.RetainedCandidates = len(summary.Modules)
	summary.CompressionRatio = float64(summary.RetainedCandidates) / float64(summary.TotalCandidates)
	return summary
}

func indexModulePaths(modules []ModuleCandidate) ([]string, map[string]int) {
	paths := make([]string, len(modules))
	index := make(map[string]int, len(modules))
	for i, module := range modules {
		path := filepath.ToSlash(module.Path)
		paths[i] = path
		index[path] = i
	}
	return paths, index
}

func moduleSubtreeFileCounts(paths []string, dirStats map[string]*dirStat) []int {
	counts := make([]int, len(paths))
	for i, path := range paths {
		for dir, stats := range dirStats {
			normalizedDir := filepath.ToSlash(dir)
			if normalizedDir == path || strings.HasPrefix(normalizedDir, path+"/") {
				counts[i] += stats.FileCount
			}
		}
	}
	return counts
}

func moduleParents(paths []string, index map[string]int) []int {
	parents := make([]int, len(paths))
	for i := range parents {
		parents[i] = -1
	}

	for i, path := range paths {
		parentPath := path
		for {
			if parentPath == "_root" || parentPath == "." || parentPath == "" {
				break
			}
			if separator := strings.LastIndex(parentPath, "/"); separator == -1 {
				parentPath = "_root"
			} else {
				parentPath = parentPath[:separator]
			}
			if parent, ok := index[parentPath]; ok {
				parents[i] = parent
				break
			}
		}
	}
	return parents
}

// retainedModules walks candidates shallowest-first so a parent's retention
// decision is always settled before its children are evaluated, then keeps
// each child if it is strong relative to its parent's score or different
// enough in extension/category mix to represent a distinct subsystem rather
// than a redundant nested view of the same one.
func retainedModules(
	paths []string,
	parents []int,
	scores []moduleScore,
	profile HeuristicProfile,
) []bool {
	indexes := make([]int, len(paths))
	for i := range paths {
		indexes[i] = i
	}
	sort.Slice(indexes, func(i, j int) bool {
		leftDepth := strings.Count(paths[indexes[i]], "/")
		rightDepth := strings.Count(paths[indexes[j]], "/")
		if leftDepth == rightDepth {
			return paths[indexes[i]] < paths[indexes[j]]
		}
		return leftDepth < rightDepth
	})

	retained := make([]bool, len(paths))
	for _, i := range indexes {
		if parents[i] == -1 {
			retained[i] = true
			continue
		}

		parent := parents[i]
		if !retained[parent] {
			retained[i] = isStrongComparedToParent(scores[i].Score, scores[parent].Score, profile) ||
				isNovelComparedToParent(scores[i].CategoryOverlap, profile)
			continue
		}
		retained[i] = isStrongComparedToParent(scores[i].Score, scores[parent].Score, profile) ||
			isNovelComparedToParent(scores[i].ExtensionOverlap, profile) ||
			isNovelComparedToParent(scores[i].CategoryOverlap, profile)
	}
	return retained
}

// isStrongComparedToParent answers "does this child retain enough of its
// parent's strength to survive on its own merits?" — at least
// ChildScoreRetentionRatio of the parent's score.
//
// That question only makes sense when there is positive parent strength to
// retain a fraction of. A parent's score goes negative once the redundant-
// child penalty (see scoreModules) exceeds its own evidence/size
// contribution — i.e. the parent itself already looks redundant. Multiplying
// a negative parent score by a fractional ratio makes it LESS negative, not
// smaller in magnitude, so the raw formula inverts: the more redundant the
// parent, the easier the bar is to clear, letting an equally redundant child
// look "strong" by comparison. Clamping the parent side at zero treats a
// non-positive parent as having no strength to be a fraction of, so a child
// can only pass this check by having a genuinely non-negative score of its
// own — it can no longer pass by being merely less negative than a parent
// that is itself redundant.
func isStrongComparedToParent(childScore, parentScore int, profile HeuristicProfile) bool {
	effectiveParentScore := parentScore
	if effectiveParentScore < 0 {
		effectiveParentScore = 0
	}
	return float64(childScore) >= float64(effectiveParentScore)*profile.Compression.ChildScoreRetentionRatio
}

func isNovelComparedToParent(overlap float64, profile HeuristicProfile) bool {
	return (1.0 - overlap) > profile.Compression.NoveltyRetentionDelta
}
