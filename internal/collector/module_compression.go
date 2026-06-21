package collector

import (
	"path/filepath"
	"sort"
	"strings"
)

func compressModules(
	modules []ModuleCandidate,
	dirStats map[string]*dirStat,
	totalFiles int,
) CompressedModuleSummary {
	summary := CompressedModuleSummary{TotalCandidates: len(modules)}
	if len(modules) == 0 {
		return summary
	}

	normalizedPaths, index := indexModulePaths(modules)
	subtreeFiles := moduleSubtreeFileCounts(normalizedPaths, dirStats)
	parents := moduleParents(normalizedPaths, index)
	scores, extensionOverlap, categoryOverlap := scoreModules(
		modules,
		subtreeFiles,
		parents,
		totalFiles,
	)
	retained := retainedModules(normalizedPaths, parents, scores, extensionOverlap, categoryOverlap)

	for i, keep := range retained {
		if !keep {
			continue
		}
		module := modules[i]
		module.Score = scores[i]
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

func retainedModules(
	paths []string,
	parents []int,
	scores []int,
	extensionOverlap []float64,
	categoryOverlap []float64,
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
			retained[i] = isStrongComparedToParent(scores[i], scores[parent]) ||
				isNovelComparedToParent(categoryOverlap[i])
			continue
		}
		retained[i] = isStrongComparedToParent(scores[i], scores[parent]) ||
			isNovelComparedToParent(extensionOverlap[i]) ||
			isNovelComparedToParent(categoryOverlap[i])
	}
	return retained
}

func isStrongComparedToParent(childScore, parentScore int) bool {
	return float64(childScore) >= float64(parentScore)*childScoreRetentionRatio
}

func isNovelComparedToParent(overlap float64) bool {
	return (1.0 - overlap) > noveltyRetentionDelta
}
