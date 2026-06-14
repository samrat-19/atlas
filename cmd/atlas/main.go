package main

import (
	"fmt"
	"os"
	"sort"

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

	// Print the results. Using `fmt.Printf` for simple, human-readable output.
	fmt.Println("Root path:", res.Root)
	fmt.Println("Total file count:", res.TotalFiles)
	printEvidenceSummary(res)
}

func printEvidenceSummary(res collector.Result) {
	fmt.Println()
	if len(res.Evidence) == 0 {
		fmt.Println("Evidence Found: none")
		return
	}

	fmt.Println("Evidence Found:")
	seen := make(map[string]string)
	for _, e := range res.Evidence {
		if _, ok := seen[e.Filename]; !ok {
			seen[e.Filename] = e.Category
		}
	}
	for fn, cat := range seen {
		fmt.Printf("- %s (%s)\n", fn, cat)
	}

	fmt.Println()
	fmt.Println("Evidence counts by category:")
	for category, count := range res.TopologySummary.EvidenceCountByCategory {
		fmt.Printf("- %s: %d\n", category, count)
	}

	fmt.Println()
	fmt.Println("Evidence counts by filename:")
	for filename, count := range res.TopologySummary.EvidenceCountByFilename {
		fmt.Printf("- %s: %d\n", filename, count)
	}

	fmt.Println()
	fmt.Printf("Root-level evidence count: %d\n", res.TopologySummary.RootEvidenceCount)
	fmt.Printf("Nested evidence count: %d\n", res.TopologySummary.NestedEvidenceCount)

	fmt.Println()
	printTopClusters(res.ClusterSummary)

	fmt.Println()
	printTopDirectories(res.CensusSummary)

	fmt.Println()
	printTopExtensions(res.ExtensionSummary, res.ClusterSummary)
}

func printTopExtensions(exts collector.ExtensionSummary, clusters collector.ClusterSummary) {
	if len(exts.ByExtension) == 0 {
		fmt.Println("Top Extensions: none")
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

	fmt.Println("Top Extensions:")
	for i, p := range pairs {
		if i >= 10 {
			break
		}
		label := p.Ext
		if label == "" {
			label = "(no ext)"
		}
		fmt.Printf("- %s: %d files\n", label, p.Count)
	}

	// Per-cluster top extensions
	fmt.Println()
	fmt.Println("Top Extensions Per Cluster:")
	// iterate clusters in deterministic order
	names := make([]string, 0, len(clusters.Clusters))
	for name := range clusters.Clusters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s:\n", name)
		m := exts.ByCluster[name]
		if len(m) == 0 {
			fmt.Println("  none")
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
			fmt.Printf("  - %s: %d\n", label, p.Count)
		}
	}
}

func printTopDirectories(census collector.CensusSummary) {
	if len(census.Directories) == 0 {
		fmt.Println("Top Directories: none")
		return
	}

	directories := make([]collector.DirectoryCensus, 0, len(census.Directories))
	for _, dir := range census.Directories {
		directories = append(directories, dir)
	}

	sort.Slice(directories, func(i, j int) bool {
		return directories[i].TotalFiles > directories[j].TotalFiles
	})

	fmt.Println("Top Directories:")
	for i, dir := range directories {
		if i >= 10 {
			break
		}
		fmt.Printf("- %s (%d files, %d evidence)\n", dir.DirectoryName, dir.TotalFiles, dir.EvidenceItemCount)
	}
}

func printTopClusters(clusterSummary collector.ClusterSummary) {
	if len(clusterSummary.Clusters) == 0 {
		fmt.Println("Top Clusters: none")
		return
	}

	clusters := make([]collector.EvidenceCluster, 0, len(clusterSummary.Clusters))
	for _, cluster := range clusterSummary.Clusters {
		clusters = append(clusters, cluster)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].EvidenceItemCount > clusters[j].EvidenceItemCount
	})

	fmt.Println("Top Clusters:")
	for i, cluster := range clusters {
		if i >= 10 {
			break
		}
		fmt.Printf("- %s (%d evidence files)\n", cluster.ClusterName, cluster.EvidenceItemCount)
		fmt.Println("  category breakdown:")
		for category, count := range cluster.EvidenceCountByCategory {
			fmt.Printf("  - %s: %d\n", category, count)
		}
	}
}
