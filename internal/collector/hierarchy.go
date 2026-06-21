package collector

import (
	"sort"
	"strings"
)

func buildHierarchy(modules []ModuleCandidate) HierarchySummary {
	if len(modules) == 0 {
		return HierarchySummary{}
	}

	paths, index := indexModulePaths(modules)
	parents := hierarchyParents(paths, index)
	nodes := hierarchyNodes(modules, paths)
	regions := attachHierarchyNodes(modules, paths, parents, nodes)

	result := make([]*RegionNode, 0, len(regions))
	for _, region := range regions {
		aggregateRegion(region)
		sortRegionTree(region)
		value := copyRegionNode(region)
		result = append(result, &value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			if result[i].FileCount == result[j].FileCount {
				if result[i].EvidenceCount == result[j].EvidenceCount {
					return result[i].Path < result[j].Path
				}
				return result[i].EvidenceCount > result[j].EvidenceCount
			}
			return result[i].FileCount > result[j].FileCount
		}
		return result[i].Score > result[j].Score
	})

	return HierarchySummary{
		TotalRegions:    len(result),
		TotalSubsystems: countSubsystems(result),
		Regions:         result,
	}
}

func hierarchyParents(paths []string, index map[string]int) []int {
	parents := make([]int, len(paths))
	for i := range parents {
		parents[i] = -1
	}
	for i, path := range paths {
		parentPath := path
		for {
			separator := strings.LastIndex(parentPath, "/")
			if separator == -1 {
				break
			}
			parentPath = parentPath[:separator]
			if parent, ok := index[parentPath]; ok {
				parents[i] = parent
				break
			}
		}
	}
	return parents
}

func hierarchyNodes(modules []ModuleCandidate, paths []string) map[string]*RegionNode {
	nodes := make(map[string]*RegionNode, len(modules))
	for i, module := range modules {
		nodes[paths[i]] = &RegionNode{
			Path:          module.Path,
			FileCount:     module.FileCount,
			EvidenceCount: module.EvidenceCount,
			Score:         module.Score,
		}
	}
	return nodes
}

func attachHierarchyNodes(
	modules []ModuleCandidate,
	paths []string,
	parents []int,
	nodes map[string]*RegionNode,
) map[string]*RegionNode {
	regions := make(map[string]*RegionNode)
	for i, module := range modules {
		path := paths[i]
		regionKey := strings.Split(path, "/")[0]
		region, ok := regions[regionKey]
		if !ok {
			region = nodes[regionKey]
			if region == nil {
				region = &RegionNode{Path: regionKey}
			}
			regions[regionKey] = region
		}

		if parents[i] == -1 {
			if path == regionKey {
				region.FileCount = module.FileCount
				region.EvidenceCount = module.EvidenceCount
				region.Score = module.Score
			} else {
				region.Children = append(region.Children, nodes[path])
			}
			continue
		}
		parentPath := paths[parents[i]]
		nodes[parentPath].Children = append(nodes[parentPath].Children, nodes[path])
	}
	return regions
}
