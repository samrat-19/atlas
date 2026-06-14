package main

import (
	"fmt"
	"os"

	"atlas/internal/collector"
)

func main() {
	root, err := scanRoot(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get current directory:", err)
		os.Exit(1)
	}

	result, err := collector.Collect(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collection failed:", err)
		os.Exit(2)
	}

	report := renderReport(result)
	fmt.Print(report)
	writeReports(result, report)
}

func scanRoot(args []string) (string, error) {
	if len(args) > 1 && args[1] != "" {
		return args[1], nil
	}
	return os.Getwd()
}
