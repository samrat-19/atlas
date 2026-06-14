package collector

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Collect walks the directory tree rooted at `root` and counts files and
// discovers evidence items as defined in the EvidenceRegistry. It performs a
// single filesystem walk and returns the aggregated Result.
func Collect(root string) (Result, error) {
	var res Result

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return res, err
	}
	res.Root = absRoot

	count := 0
	var evidence []EvidenceItem
	summary := TopologySummary{
		EvidenceCountByCategory: make(map[string]int),
		EvidenceCountByFilename: make(map[string]int),
	}
	clusterSummary := ClusterSummary{
		Clusters: make(map[string]EvidenceCluster),
	}
	censusSummary := CensusSummary{
		Directories: make(map[string]DirectoryCensus),
	}
	extensionSummary := ExtensionSummary{
		ByExtension: make(map[string]int),
		ByCluster:   make(map[string]map[string]int),
	}
	// per-directory statistics (keyed by relative directory path)
	type dirStat struct {
		FileCount          int
		EvidenceCount      int
		Extensions         map[string]int
		EvidenceByCategory map[string]int
		EvidenceByFilename map[string]int
	}
	dirStats := make(map[string]*dirStat)

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			count++
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absRoot, absPath)
		if err != nil {
			return err
		}

		topDir := topLevelDirectoryForRelativePath(relPath)
		if !d.IsDir() {
			censusSummary.updateWithFile(topDir)
			// collect extension stats
			ext := strings.ToLower(filepath.Ext(d.Name()))
			// filepath.Ext returns "" for files without extensions; we keep that
			extensionSummary.ByExtension[ext]++
			// ensure cluster map exists
			if _, ok := extensionSummary.ByCluster[topDir]; !ok {
				extensionSummary.ByCluster[topDir] = make(map[string]int)
			}
			extensionSummary.ByCluster[topDir][ext]++
			// update dirStats for the file's directory
			dir := filepath.Dir(relPath)
			if dir == "." {
				dir = "_root"
			}
			ds, ok := dirStats[dir]
			if !ok {
				ds = &dirStat{
					Extensions:         make(map[string]int),
					EvidenceByCategory: make(map[string]int),
					EvidenceByFilename: make(map[string]int),
				}
				dirStats[dir] = ds
			}
			ds.FileCount++
			ds.Extensions[ext]++
			// (no-op) previously collected sample files for UI preview; removed.
		}

		if item, ok := createEvidenceItem(d, absPath, relPath); ok {
			evidence = append(evidence, item)
			summary.updateWithEvidence(item)
			clusterSummary.updateWithEvidence(item)
			censusSummary.updateWithEvidence(topDir)
			// update dirStats for evidence
			dir := filepath.Dir(relPath)
			if dir == "." {
				dir = "_root"
			}
			ds, ok := dirStats[dir]
			if !ok {
				ds = &dirStat{
					Extensions:         make(map[string]int),
					EvidenceByCategory: make(map[string]int),
					EvidenceByFilename: make(map[string]int),
				}
				dirStats[dir] = ds
			}
			ds.EvidenceCount++
			ds.EvidenceByCategory[item.Category]++
			ds.EvidenceByFilename[item.Filename]++
		}
		return nil
	}

	if err := filepath.WalkDir(absRoot, walkFn); err != nil {
		return res, err
	}

	censusSummary.TotalDirectories = len(censusSummary.Directories)
	res.TotalFiles = count
	res.Evidence = evidence
	res.TopologySummary = summary
	res.ClusterSummary = clusterSummary
	res.CensusSummary = censusSummary
	res.ExtensionSummary = extensionSummary
	// Build module candidates from dirStats.
	var modules []ModuleCandidate
	for path, ds := range dirStats {
		// Heuristics for candidate selection:
		// - directory has any evidence
		// - or large file count
		// - or non-trivial evidence density
		density := 0.0
		if ds.FileCount > 0 {
			density = float64(ds.EvidenceCount) / float64(ds.FileCount)
		}
		if ds.EvidenceCount > 0 || ds.FileCount >= 200 || (ds.FileCount >= 20 && density >= 0.05) {
			// compute top extensions (top 3)
			type pair struct {
				e string
				c int
			}
			ps := make([]pair, 0, len(ds.Extensions))
			for e, c := range ds.Extensions {
				ps = append(ps, pair{e, c})
			}
			sort.Slice(ps, func(i, j int) bool { return ps[i].c > ps[j].c })
			topExt := make([]string, 0, 3)
			for i := 0; i < len(ps) && i < 3; i++ {
				topExt = append(topExt, ps[i].e)
			}

			mc := ModuleCandidate{
				Path:               path,
				FileCount:          ds.FileCount,
				EvidenceCount:      ds.EvidenceCount,
				DominantExtensions: topExt,
				EvidenceByCategory: copyMap(ds.EvidenceByCategory),
				EvidenceByFilename: copyMap(ds.EvidenceByFilename),
			}
			modules = append(modules, mc)
		}
	}
	// score and sort modules by confidence (evidence-heavy first, then size)
	type scored struct {
		m     ModuleCandidate
		score int
	}
	scs := make([]scored, 0, len(modules))
	for _, m := range modules {
		score := m.EvidenceCount*100 + m.FileCount
		scs = append(scs, scored{m, score})
	}
	sort.Slice(scs, func(i, j int) bool { return scs[i].score > scs[j].score })
	modList := make([]ModuleCandidate, 0, len(scs))
	for _, s := range scs {
		modList = append(modList, s.m)
	}
	res.ModuleSummary = ModuleSummary{TotalModules: len(modList), Modules: modList}

	// Compress module candidates into a smaller set of high-value modules.
	compressed := func(mods []ModuleCandidate, dirStats map[string]*dirStat, totalFiles int) CompressedModuleSummary {
		cm := CompressedModuleSummary{}
		total := len(mods)
		cm.TotalCandidates = total
		if total == 0 {
			cm.RetainedCandidates = 0
			cm.CompressionRatio = 0
			cm.Modules = nil
			return cm
		}

		// normalize module paths and build index
		index := make(map[string]int)
		norms := make([]string, len(mods))
		for i, m := range mods {
			n := filepath.ToSlash(m.Path)
			norms[i] = n
			index[n] = i
		}

		// helper to compute subtree file count for a candidate
		subtreeFiles := make([]int, len(mods))
		for i, n := range norms {
			sum := 0
			for dk, ds := range dirStats {
				dn := filepath.ToSlash(dk)
				if dn == n || strings.HasPrefix(dn, n+"/") {
					sum += ds.FileCount
				}
			}
			subtreeFiles[i] = sum
		}

		// find parent for each candidate (nearest ancestor candidate)
		parents := make([]int, len(mods))
		for i := range parents {
			parents[i] = -1
		}
		for i, n := range norms {
			p := n
			for {
				if p == "_root" || p == "." || p == "" {
					break
				}
				// trim last path segment
				idx := strings.LastIndex(p, "/")
				if idx == -1 {
					p = "_root"
				} else {
					p = p[:idx]
				}
				if j, ok := index[p]; ok {
					parents[i] = j
					break
				}
			}
		}

		// compute overlaps and base scores
		scores := make([]int, len(mods))
		extOverlap := make([]float64, len(mods))
		catOverlap := make([]float64, len(mods))
		for i, m := range mods {
			// coverage percent
			coverage := 0
			if totalFiles > 0 {
				coverage = subtreeFiles[i] * 100 / totalFiles
			}
			coverageScore := coverage * 10
			// base score: weight evidence strongly
			base := m.EvidenceCount*200 + m.FileCount + coverageScore

			// compute overlap with parent
			if parents[i] != -1 {
				p := mods[parents[i]]
				// extension overlap (Jaccard-like)
				extSet := make(map[string]struct{})
				for _, e := range m.DominantExtensions {
					extSet[e] = struct{}{}
				}
				pSet := make(map[string]struct{})
				for _, e := range p.DominantExtensions {
					pSet[e] = struct{}{}
				}
				inter := 0
				union := 0
				for k := range extSet {
					if _, ok := pSet[k]; ok {
						inter++
					}
				}
				union = len(extSet)
				if len(pSet) > union {
					union = len(pSet)
				}
				if union == 0 {
					extOverlap[i] = 0
				} else {
					extOverlap[i] = float64(inter) / float64(union)
				}

				// category overlap
				pcat := p.EvidenceByCategory
				interC := 0
				unionC := 0
				cset := make(map[string]struct{})
				for k := range m.EvidenceByCategory {
					cset[k] = struct{}{}
				}
				for k := range pcat {
					if _, ok := cset[k]; ok {
						interC++
					}
				}
				unionC = len(m.EvidenceByCategory)
				if len(pcat) > unionC {
					unionC = len(pcat)
				}
				if unionC == 0 {
					catOverlap[i] = 0
				} else {
					catOverlap[i] = float64(interC) / float64(unionC)
				}
			} else {
				extOverlap[i] = 0
				catOverlap[i] = 0
			}

			score := base
			// novelty penalty
			if parents[i] != -1 && extOverlap[i] >= 0.9 && catOverlap[i] >= 0.9 {
				score -= 500
			}
			scores[i] = score
		}

		// determine retention: process parents before children
		// compute depths
		idxs := make([]int, 0, len(mods))
		for i := range mods {
			idxs = append(idxs, i)
		}
		sort.Slice(idxs, func(i, j int) bool { return strings.Count(norms[idxs[i]], "/") < strings.Count(norms[idxs[j]], "/") })

		retained := make([]bool, len(mods))
		for _, i := range idxs {
			if parents[i] == -1 {
				retained[i] = true
				continue
			}
			pidx := parents[i]
			pRet := retained[pidx]
			if !pRet {
				// if parent not retained, keep child if score high enough
				if float64(scores[i]) >= float64(scores[pidx])*0.6 || (1.0-catOverlap[i]) > 0.2 {
					retained[i] = true
				}
			} else {
				// parent retained: keep child only if it adds novelty or has comparable score
				if float64(scores[i]) >= float64(scores[pidx])*0.6 || (1.0-extOverlap[i]) > 0.2 || (1.0-catOverlap[i]) > 0.2 {
					retained[i] = true
				}
			}
		}

		// build retained module list with scores
		for i, keep := range retained {
			if keep {
				m := mods[i]
				m.Score = scores[i]
				cm.Modules = append(cm.Modules, m)
			}
		}

		cm.RetainedCandidates = len(cm.Modules)
		if cm.TotalCandidates > 0 {
			cm.CompressionRatio = float64(cm.RetainedCandidates) / float64(cm.TotalCandidates)
		} else {
			cm.CompressionRatio = 0
		}
		return cm
	}(modList, dirStats, res.TotalFiles)

	res.CompressedModuleSummary = compressed
	res.HierarchySummary = buildHierarchy(res.CompressedModuleSummary.Modules)
	return res, nil
}

func copyMap(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func buildHierarchy(mods []ModuleCandidate) HierarchySummary {
	hs := HierarchySummary{}
	if len(mods) == 0 {
		return hs
	}

	index := make(map[string]int, len(mods))
	norms := make([]string, len(mods))
	for i, m := range mods {
		n := filepath.ToSlash(m.Path)
		norms[i] = n
		index[n] = i
	}

	parents := make([]int, len(mods))
	for i := range parents {
		parents[i] = -1
	}
	for i, n := range norms {
		p := n
		for {
			idx := strings.LastIndex(p, "/")
			if idx == -1 {
				break
			}
			p = p[:idx]
			if j, ok := index[p]; ok {
				parents[i] = j
				break
			}
		}
	}

	nodeMap := make(map[string]*RegionNode, len(mods))
	topRegions := make(map[string]*RegionNode)

	for i, m := range mods {
		path := norms[i]
		nodeMap[path] = &RegionNode{
			Path:          m.Path,
			FileCount:     m.FileCount,
			EvidenceCount: m.EvidenceCount,
			Score:         m.Score,
		}
	}

	for i, m := range mods {
		path := norms[i]
		regionKey := strings.Split(path, "/")[0]
		region, ok := topRegions[regionKey]
		if !ok {
			region = nodeMap[regionKey]
			if region == nil {
				region = &RegionNode{Path: regionKey}
			}
			topRegions[regionKey] = region
		}

		if parents[i] == -1 {
			if path == regionKey {
				region.FileCount = m.FileCount
				region.EvidenceCount = m.EvidenceCount
				region.Score = m.Score
			} else {
				region.Children = append(region.Children, nodeMap[path])
			}
		} else {
			parentPath := norms[parents[i]]
			parent := nodeMap[parentPath]
			parent.Children = append(parent.Children, nodeMap[path])
		}
	}

	regions := make([]*RegionNode, 0, len(topRegions))
	for _, region := range topRegions {
		aggregateRegion(region)
		sortRegionTree(region)
		regionValue := toValueNode(region)
		regions = append(regions, &regionValue)
	}

	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Score > regions[j].Score
	})

	hs.Regions = regions
	hs.TotalRegions = len(regions)
	hs.TotalSubsystems = countSubsystems(regions)
	return hs
}

func aggregateRegion(region *RegionNode) {
	if region.FileCount == 0 && region.Score == 0 {
		totalFiles := 0
		totalEvidence := 0
		topScore := 0
		for _, child := range region.Children {
			totalFiles += child.FileCount
			totalEvidence += child.EvidenceCount
			if child.Score > topScore {
				topScore = child.Score
			}
		}
		region.FileCount = totalFiles
		region.EvidenceCount = totalEvidence
		region.Score = topScore
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
	if len(node.Children) > 0 {
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
}

func sortRegionTree(node *RegionNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].Score == node.Children[j].Score {
			return node.Children[i].FileCount > node.Children[j].FileCount
		}
		return node.Children[i].Score > node.Children[j].Score
	})
	for _, child := range node.Children {
		sortRegionTree(child)
	}
}

func toValueNode(node *RegionNode) RegionNode {
	children := make([]*RegionNode, 0, len(node.Children))
	for _, child := range node.Children {
		childValue := toValueNode(child)
		children = append(children, &childValue)
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

func createEvidenceItem(d fs.DirEntry, absPath, relPath string) (EvidenceItem, bool) {
	// Use the registry matcher which handles normalization and indexing.
	if key, category, ok := MatchEvidence(d.Name(), relPath); ok {
		return EvidenceItem{Filename: key, AbsolutePath: absPath, RelativePath: relPath, Category: category}, true
	}
	return EvidenceItem{}, false
}

func (s *TopologySummary) updateWithEvidence(item EvidenceItem) {
	s.TotalEvidenceItems++
	s.EvidenceCountByCategory[item.Category]++
	s.EvidenceCountByFilename[item.Filename]++

	if filepath.Dir(item.RelativePath) == "." {
		s.RootEvidenceCount++
	} else {
		s.NestedEvidenceCount++
	}
}

func (s *ClusterSummary) updateWithEvidence(item EvidenceItem) {
	clusterName := topLevelDirectoryForRelativePath(item.RelativePath)
	cluster, ok := s.Clusters[clusterName]
	if !ok {
		cluster = EvidenceCluster{
			ClusterName:             clusterName,
			EvidenceCountByCategory: make(map[string]int),
			EvidenceCountByFilename: make(map[string]int),
		}
	}

	cluster.EvidenceItemCount++
	cluster.EvidenceCountByCategory[item.Category]++
	cluster.EvidenceCountByFilename[item.Filename]++

	s.Clusters[clusterName] = cluster
}

func (s *CensusSummary) updateWithFile(directory string) {
	entry, ok := s.Directories[directory]
	if !ok {
		entry = DirectoryCensus{DirectoryName: directory}
	}
	entry.TotalFiles++
	s.Directories[directory] = entry
}

func (s *CensusSummary) updateWithEvidence(directory string) {
	entry, ok := s.Directories[directory]
	if !ok {
		entry = DirectoryCensus{DirectoryName: directory}
	}
	entry.EvidenceItemCount++
	s.Directories[directory] = entry
}

func topLevelDirectoryForRelativePath(relPath string) string {
	normRelPath := filepath.ToSlash(relPath)
	parts := strings.Split(normRelPath, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "_root"
	}
	if len(parts) == 1 {
		return "_root"
	}
	return parts[0]
}
