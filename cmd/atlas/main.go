package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"atlas/internal/collector"
)

// package main: an executable program must have package `main` and a `main` function.
// The `main` function is the program entry point.
func main() {
	// Determine the root path to collect. Prefer a first CLI arg, fallback to cwd.
	var root string
	if len(os.Args) > 1 && os.Args[1] != "" {
		root = os.Args[1]
	} else {
		// Get the current working directory. We'll collect stats for this directory.
		cwd, err := os.Getwd()
		if err != nil {
			// Proper error handling: print the error and exit with non-zero status.
			fmt.Fprintln(os.Stderr, "failed to get current directory:", err)
			os.Exit(1)
		}
		root = cwd
	}

	// Call the exported Collect function from the internal/collector package.
	// Note: `collector.Collect` returns a `Result` and an `error`.
	res, err := collector.Collect(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collection failed:", err)
		os.Exit(2)
	}

	// Build the textual report into a buffer, print to stdout, and save to output dir.
	var b strings.Builder
	fmt.Fprintf(&b, "Root path: %s\n", res.Root)
	fmt.Fprintf(&b, "Total file count: %d\n", res.TotalFiles)
	printEvidenceSummary(res, &b)

	report := b.String()
	// print to stdout
	fmt.Print(report)

	// write to output file
	outDir := "output"
	if err := os.MkdirAll(outDir, 0755); err == nil {
		// sanitize root for filename
		name := strings.ReplaceAll(res.Root, string(os.PathSeparator), "_")
		name = strings.ReplaceAll(name, ":", "")
		timestamp := time.Now().Format("20060102T150405")
		fname := fmt.Sprintf("%s-%s.txt", name, timestamp)
		outPath := filepath.Join(outDir, fname)
		_ = os.WriteFile(outPath, []byte(report), 0644)
		// also write a JSON and an HTML report (standalone) next to the text file
		if jb, err := json.MarshalIndent(res, "", "  "); err == nil {
			// write JSON only (no HTML report)
			_ = os.WriteFile(filepath.Join(outDir, fmt.Sprintf("%s-%s.json", name, timestamp)), jb, 0644)
		}
	}
}

func printEvidenceSummary(res collector.Result, w io.Writer) {
	fmt.Fprintln(w)
	if len(res.Evidence) == 0 {
		fmt.Fprintln(w, "Evidence Found: none")
		return
	}

	fmt.Fprintln(w, "Evidence Found:")
	seen := make(map[string]string)
	for _, e := range res.Evidence {
		if _, ok := seen[e.Filename]; !ok {
			seen[e.Filename] = e.Category
		}
	}
	for fn, cat := range seen {
		fmt.Fprintf(w, "- %s (%s)\n", fn, cat)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Evidence counts by category:")
	for category, count := range res.TopologySummary.EvidenceCountByCategory {
		fmt.Fprintf(w, "- %s: %d\n", category, count)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Evidence counts by filename:")
	for filename, count := range res.TopologySummary.EvidenceCountByFilename {
		fmt.Fprintf(w, "- %s: %d\n", filename, count)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Root-level evidence count: %d\n", res.TopologySummary.RootEvidenceCount)
	fmt.Fprintf(w, "Nested evidence count: %d\n", res.TopologySummary.NestedEvidenceCount)

	fmt.Fprintln(w)
	printTopClusters(res.ClusterSummary, w)

	fmt.Fprintln(w)
	printTopDirectories(res.CensusSummary, w)

	fmt.Fprintln(w)
	printTopExtensions(res.ExtensionSummary, res.ClusterSummary, w)

	fmt.Fprintln(w)
	printRepositoryHierarchy(res.HierarchySummary, w)

	fmt.Fprintln(w)
	printMajorModules(res.CompressedModuleSummary, w)
}

func printRepositoryHierarchy(hs collector.HierarchySummary, w io.Writer) {
	fmt.Fprintln(w, "Repository Hierarchy:")
	if len(hs.Regions) == 0 {
		fmt.Fprintln(w, "none")
		return
	}

	for _, region := range hs.Regions {
		fmt.Fprintf(w, "%s\n", region.Path)
		fmt.Fprintf(w, "files: %d\n", region.FileCount)
		if region.EvidenceCount > 0 {
			fmt.Fprintf(w, "evidence: %d\n", region.EvidenceCount)
		}
		if len(region.Children) > 0 {
			fmt.Fprintln(w, "subsystems:")
			for _, sub := range region.Children {
				fmt.Fprintf(w, "- %s\n", strings.TrimPrefix(sub.Path, region.Path+"/"))
				if len(sub.Children) > 0 {
					fmt.Fprintln(w, "  components:")
					for _, comp := range sub.Children {
						fmt.Fprintf(w, "  - %s\n", strings.TrimPrefix(comp.Path, sub.Path+"/"))
					}
				}
			}
		}
		fmt.Fprintln(w)
	}
}

func printMajorModules(cm collector.CompressedModuleSummary, w io.Writer) {
	if cm.TotalCandidates == 0 {
		fmt.Fprintln(w, "Candidate Modules Found: 0")
		return
	}
	fmt.Fprintf(w, "Candidate Modules Found: %d\n", cm.TotalCandidates)
	fmt.Fprintf(w, "Retained Modules: %d\n", cm.RetainedCandidates)
	fmt.Fprintf(w, "Compression Ratio: %.2f\n", cm.CompressionRatio)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Major Modules:")
	// sort retained modules by score desc
	s := cm.Modules
	sort.Slice(s, func(i, j int) bool { return s[i].Score > s[j].Score })
	for _, m := range s {
		fmt.Fprintf(w, "- %s\n", m.Path)
		fmt.Fprintf(w, "  score: %d\n", m.Score)
		fmt.Fprintf(w, "  files: %d\n", m.FileCount)
		fmt.Fprintf(w, "  evidence: %d\n", m.EvidenceCount)
	}
}

func printTopExtensions(exts collector.ExtensionSummary, clusters collector.ClusterSummary, w io.Writer) {
	if len(exts.ByExtension) == 0 {
		fmt.Fprintln(w, "Top Extensions: none")
		return
	}

	// Build slice of pairs for sorting
	type pair struct {
		Ext   string
		Count int
	}
	pairs := make([]pair, 0, len(exts.ByExtension))
	for e, c := range exts.ByExtension {
		pairs = append(pairs, pair{Ext: e, Count: c})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Count > pairs[j].Count })

	fmt.Fprintln(w, "Top Extensions:")
	for i, p := range pairs {
		if i >= 10 {
			break
		}
		label := p.Ext
		if label == "" {
			label = "(no ext)"
		}
		fmt.Fprintf(w, "- %s: %d files\n", label, p.Count)
	}

	// Per-cluster top extensions
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Top Extensions Per Cluster:")
	// iterate clusters in deterministic order
	names := make([]string, 0, len(clusters.Clusters))
	for name := range clusters.Clusters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(w, "%s:\n", name)
		m := exts.ByCluster[name]
		if len(m) == 0 {
			fmt.Fprintln(w, "  none")
			continue
		}
		// sort
		ps := make([]pair, 0, len(m))
		for e, c := range m {
			ps = append(ps, pair{Ext: e, Count: c})
		}
		sort.Slice(ps, func(i, j int) bool { return ps[i].Count > ps[j].Count })
		for i, p := range ps {
			if i >= 5 {
				break
			}
			label := p.Ext
			if label == "" {
				label = "(no ext)"
			}
			fmt.Fprintf(w, "  - %s: %d\n", label, p.Count)
		}
	}
}

func printTopDirectories(census collector.CensusSummary, w io.Writer) {
	if len(census.Directories) == 0 {
		fmt.Fprintln(w, "Top Directories: none")
		return
	}

	directories := make([]collector.DirectoryCensus, 0, len(census.Directories))
	for _, dir := range census.Directories {
		directories = append(directories, dir)
	}

	sort.Slice(directories, func(i, j int) bool {
		return directories[i].TotalFiles > directories[j].TotalFiles
	})

	fmt.Fprintln(w, "Top Directories:")
	for i, dir := range directories {
		if i >= 10 {
			break
		}
		fmt.Fprintf(w, "- %s (%d files, %d evidence)\n", dir.DirectoryName, dir.TotalFiles, dir.EvidenceItemCount)
	}
}

func printTopClusters(clusterSummary collector.ClusterSummary, w io.Writer) {
	if len(clusterSummary.Clusters) == 0 {
		fmt.Fprintln(w, "Top Clusters: none")
		return
	}

	clusters := make([]collector.EvidenceCluster, 0, len(clusterSummary.Clusters))
	for _, cluster := range clusterSummary.Clusters {
		clusters = append(clusters, cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].EvidenceItemCount > clusters[j].EvidenceItemCount
	})

	fmt.Fprintln(w, "Top Clusters:")
	for i, cluster := range clusters {
		if i >= 10 {
			break
		}
		fmt.Fprintf(w, "- %s (%d evidence files)\n", cluster.ClusterName, cluster.EvidenceItemCount)
		fmt.Fprintln(w, "  category breakdown:")
		for category, count := range cluster.EvidenceCountByCategory {
			fmt.Fprintf(w, "  - %s: %d\n", category, count)
		}
	}
}
