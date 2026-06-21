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

	// Output failure is a distinct condition from scan failure: the analysis
	// succeeded and was printed to stdout, but the on-disk artifacts could not
	// be written. Exit 3 signals this so callers can distinguish the two.
	if err := writeReports(result, report); err != nil {
		fmt.Fprintln(os.Stderr, "failed to write output files:", err)
		os.Exit(3)
	}
}

func scanRoot(args []string) (string, error) {
	if len(args) > 1 && args[1] != "" {
		return args[1], nil
	}
	return os.Getwd()
}
