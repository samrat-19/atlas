package collector

import "sort"

func aggregateRegion(region *RegionNode) {
	if region.FileCount == 0 && region.Score == 0 {
		for _, child := range region.Children {
			region.FileCount += child.FileCount
			region.EvidenceCount += child.EvidenceCount
			if child.Score > region.Score {
				region.Score = child.Score
			}
		}
	} else {
		totalFiles := region.FileCount
		totalEvidence := region.EvidenceCount
		topScore := region.Score
		for _, child := range region.Children {
			totalFiles += child.FileCount
			totalEvidence += child.EvidenceCount
			if child.Score > topScore {
				topScore = child.Score
			}
		}
		if totalFiles > region.FileCount {
			region.FileCount = totalFiles
		}
		if totalEvidence > region.EvidenceCount {
			region.EvidenceCount = totalEvidence
		}
		if topScore > region.Score {
			region.Score = topScore
		}
	}

	for _, child := range region.Children {
		aggregateSubtree(child)
	}
}

func aggregateSubtree(node *RegionNode) {
	for _, child := range node.Children {
		aggregateSubtree(child)
	}
	if len(node.Children) == 0 {
		return
	}

	childFiles := 0
	childEvidence := 0
	topScore := node.Score
	for _, child := range node.Children {
		childFiles += child.FileCount
		childEvidence += child.EvidenceCount
		if child.Score > topScore {
			topScore = child.Score
		}
	}
	if childFiles > node.FileCount {
		node.FileCount = childFiles
	}
	if childEvidence > node.EvidenceCount {
		node.EvidenceCount = childEvidence
	}
	if topScore > node.Score {
		node.Score = topScore
	}
}

func sortRegionTree(node *RegionNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].Score == node.Children[j].Score {
			if node.Children[i].FileCount == node.Children[j].FileCount {
				if node.Children[i].EvidenceCount == node.Children[j].EvidenceCount {
					return node.Children[i].Path < node.Children[j].Path
				}
				return node.Children[i].EvidenceCount > node.Children[j].EvidenceCount
			}
			return node.Children[i].FileCount > node.Children[j].FileCount
		}
		return node.Children[i].Score > node.Children[j].Score
	})
	for _, child := range node.Children {
		sortRegionTree(child)
	}
}

func copyRegionNode(node *RegionNode) RegionNode {
	children := make([]*RegionNode, 0, len(node.Children))
	for _, child := range node.Children {
		value := copyRegionNode(child)
		children = append(children, &value)
	}
	return RegionNode{
		Path:          node.Path,
		FileCount:     node.FileCount,
		EvidenceCount: node.EvidenceCount,
		Score:         node.Score,
		Children:      children,
	}
}

func countSubsystems(regions []*RegionNode) int {
	count := 0
	for _, region := range regions {
		count += countNodes(region.Children)
	}
	return count
}

func countNodes(nodes []*RegionNode) int {
	count := len(nodes)
	for _, node := range nodes {
		count += countNodes(node.Children)
	}
	return count
}
