package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"atlas/internal/collector"
)

func printRepositoryHierarchy(summary collector.HierarchySummary, writer io.Writer) {
	fmt.Fprintln(writer, "Repository Hierarchy:")
	if len(summary.Regions) == 0 {
		fmt.Fprintln(writer, "none")
		return
	}

	for _, region := range summary.Regions {
		fmt.Fprintf(writer, "%s\n", region.Path)
		fmt.Fprintf(writer, "files: %d\n", region.FileCount)
		if region.EvidenceCount > 0 {
			fmt.Fprintf(writer, "evidence: %d\n", region.EvidenceCount)
		}
		if len(region.Children) > 0 {
			fmt.Fprintln(writer, "subsystems:")
			for _, subsystem := range region.Children {
				fmt.Fprintf(writer, "- %s\n", strings.TrimPrefix(subsystem.Path, region.Path+"/"))
				if len(subsystem.Children) > 0 {
					fmt.Fprintln(writer, "  components:")
					for _, component := range subsystem.Children {
						fmt.Fprintf(writer, "  - %s\n", strings.TrimPrefix(component.Path, subsystem.Path+"/"))
					}
				}
			}
		}
		fmt.Fprintln(writer)
	}
}

func printMajorModules(summary collector.CompressedModuleSummary, writer io.Writer) {
	if summary.TotalCandidates == 0 {
		fmt.Fprintln(writer, "Candidate Modules Found: 0")
		return
	}

	fmt.Fprintf(writer, "Candidate Modules Found: %d\n", summary.TotalCandidates)
	fmt.Fprintf(writer, "Retained Modules: %d\n", summary.RetainedCandidates)
	fmt.Fprintf(writer, "Compression Ratio: %.2f\n", summary.CompressionRatio)
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Major Modules:")

	modules := summary.Modules
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Score > modules[j].Score
	})
	for _, module := range modules {
		fmt.Fprintf(writer, "- %s\n", module.Path)
		fmt.Fprintf(writer, "  score: %d\n", module.Score)
		fmt.Fprintf(writer, "  files: %d\n", module.FileCount)
		fmt.Fprintf(writer, "  evidence: %d\n", module.EvidenceCount)
	}
}

func printTopDirectories(summary collector.CensusSummary, writer io.Writer) {
	if len(summary.Directories) == 0 {
		fmt.Fprintln(writer, "Top Directories: none")
		return
	}

	directories := make([]collector.DirectoryCensus, 0, len(summary.Directories))
	for _, directory := range summary.Directories {
		directories = append(directories, directory)
	}
	sort.Slice(directories, func(i, j int) bool {
		return directories[i].TotalFiles > directories[j].TotalFiles
	})

	fmt.Fprintln(writer, "Top Directories:")
	for i, directory := range directories {
		if i >= 10 {
			break
		}
		fmt.Fprintf(
			writer,
			"- %s (%d files, %d evidence)\n",
			directory.DirectoryName,
			directory.TotalFiles,
			directory.EvidenceItemCount,
		)
	}
}

func printTopClusters(summary collector.ClusterSummary, writer io.Writer) {
	if len(summary.Clusters) == 0 {
		fmt.Fprintln(writer, "Top Clusters: none")
		return
	}

	clusters := make([]collector.EvidenceCluster, 0, len(summary.Clusters))
	for _, cluster := range summary.Clusters {
		clusters = append(clusters, cluster)
	}
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].EvidenceItemCount > clusters[j].EvidenceItemCount
	})

	fmt.Fprintln(writer, "Top Clusters:")
	for i, cluster := range clusters {
		if i >= 10 {
			break
		}
		fmt.Fprintf(writer, "- %s (%d evidence files)\n", cluster.ClusterName, cluster.EvidenceItemCount)
		fmt.Fprintln(writer, "  category breakdown:")
		for category, count := range cluster.EvidenceCountByCategory {
			fmt.Fprintf(writer, "  - %s: %d\n", category, count)
		}
	}
}
