package collector

import (
	"sort"
	"strings"

	"atlas/internal/model"
)

func buildHierarchy(modules []model.ModuleCandidate) model.HierarchySummary {
	if len(modules) == 0 {
		return model.HierarchySummary{}
	}

	paths, index := indexModulePaths(modules)
	parents := hierarchyParents(paths, index)
	nodes := hierarchyNodes(modules, paths)
	regions := attachHierarchyNodes(modules, paths, parents, nodes)

	result := make([]*model.RegionNode, 0, len(regions))
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

	return model.HierarchySummary{
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

func hierarchyNodes(modules []model.ModuleCandidate, paths []string) map[string]*model.RegionNode {
	nodes := make(map[string]*model.RegionNode, len(modules))
	for i, module := range modules {
		nodes[paths[i]] = &model.RegionNode{
			Path:          module.Path,
			FileCount:     module.FileCount,
			EvidenceCount: module.EvidenceCount,
			Score:         module.Score,
		}
	}
	return nodes
}

func attachHierarchyNodes(
	modules []model.ModuleCandidate,
	paths []string,
	parents []int,
	nodes map[string]*model.RegionNode,
) map[string]*model.RegionNode {
	regions := make(map[string]*model.RegionNode)
	for i, module := range modules {
		path := paths[i]
		regionKey := strings.Split(path, "/")[0]
		region, ok := regions[regionKey]
		if !ok {
			region = nodes[regionKey]
			if region == nil {
				region = &model.RegionNode{Path: regionKey}
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
