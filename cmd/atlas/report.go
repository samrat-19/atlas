package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"atlas/internal/collector"
)

func renderReport(result collector.Result) string {
	var report strings.Builder
	fmt.Fprintf(&report, "Root path: %s\n", result.Root)
	fmt.Fprintf(&report, "Total file count: %d\n", result.TotalFiles)
	printEvidenceSummary(result, &report)
	return report.String()
}

func printEvidenceSummary(result collector.Result, writer io.Writer) {
	fmt.Fprintln(writer)
	if len(result.Evidence) == 0 {
		fmt.Fprintln(writer, "Evidence Found: none")
		return
	}

	printEvidenceFound(result.Evidence, writer)

	fmt.Fprintln(writer)
	printCountMap("Evidence counts by category:", result.TopologySummary.EvidenceCountByCategory, writer)

	fmt.Fprintln(writer)
	printCountMap("Evidence counts by filename:", result.TopologySummary.EvidenceCountByFilename, writer)

	fmt.Fprintln(writer)
	fmt.Fprintf(writer, "Root-level evidence count: %d\n", result.TopologySummary.RootEvidenceCount)
	fmt.Fprintf(writer, "Nested evidence count: %d\n", result.TopologySummary.NestedEvidenceCount)

	fmt.Fprintln(writer)
	printTopClusters(result.ClusterSummary, writer)

	fmt.Fprintln(writer)
	printTopDirectories(result.CensusSummary, writer)

	fmt.Fprintln(writer)
	printTopExtensions(result.ExtensionSummary, result.ClusterSummary, writer)

	fmt.Fprintln(writer)
	printRepositoryHierarchy(result.HierarchySummary, writer)

	fmt.Fprintln(writer)
	printMajorModules(result.CompressedModuleSummary, writer)
}

func printEvidenceFound(evidence []collector.EvidenceItem, writer io.Writer) {
	fmt.Fprintln(writer, "Evidence Found:")
	seen := make(map[string]string)
	for _, item := range evidence {
		if _, ok := seen[item.Filename]; !ok {
			seen[item.Filename] = item.Category
		}
	}
	filenames := make([]string, 0, len(seen))
	for filename := range seen {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		fmt.Fprintf(writer, "- %s (%s)\n", filename, seen[filename])
	}
}

func printCountMap(title string, counts map[string]int, writer io.Writer) {
	fmt.Fprintln(writer, title)
	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		fmt.Fprintf(writer, "- %s: %d\n", label, counts[label])
	}
}
