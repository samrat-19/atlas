package main

import (
	"fmt"
	"io"
	"sort"

	"atlas/internal/collector"
)

type extensionCount struct {
	extension string
	count     int
}

func printTopExtensions(
	extensions collector.ExtensionSummary,
	clusters collector.ClusterSummary,
	writer io.Writer,
) {
	if len(extensions.ByExtension) == 0 {
		fmt.Fprintln(writer, "Top Extensions: none")
		return
	}

	fmt.Fprintln(writer, "Top Extensions:")
	printExtensionCounts(sortedExtensionCounts(extensions.ByExtension), topExtensionLimit, "", writer)

	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Top Extensions Per Cluster:")
	names := make([]string, 0, len(clusters.Clusters))
	for name := range clusters.Clusters {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(writer, "%s:\n", name)
		counts := extensions.ByCluster[name]
		if len(counts) == 0 {
			fmt.Fprintln(writer, "  none")
			continue
		}
		printExtensionCounts(sortedExtensionCounts(counts), topClusterExtensionLimit, "  ", writer)
	}
}

func sortedExtensionCounts(counts map[string]int) []extensionCount {
	result := make([]extensionCount, 0, len(counts))
	for extension, count := range counts {
		result = append(result, extensionCount{extension: extension, count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].count == result[j].count {
			return result[i].extension < result[j].extension
		}
		return result[i].count > result[j].count
	})
	return result
}

func printExtensionCounts(
	counts []extensionCount,
	limit int,
	indent string,
	writer io.Writer,
) {
	for i, item := range counts {
		if i >= limit {
			break
		}
		label := item.extension
		if label == "" {
			label = "(no ext)"
		}
		if indent == "" {
			fmt.Fprintf(writer, "- %s: %d files\n", label, item.count)
		} else {
			fmt.Fprintf(writer, "%s- %s: %d\n", indent, label, item.count)
		}
	}
}
