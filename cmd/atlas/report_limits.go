package main

const (
	// Show this many extensions for the whole repository. This keeps the report
	// short while still showing the main language/content types.
	topExtensionLimit = 10

	// Show this many extensions inside each top-level cluster. This is lower than
	// the repo-wide limit because there can be many clusters.
	topClusterExtensionLimit = 5

	// Show this many top-level directories in the text report. This only affects
	// display, not analysis.
	topDirectoryLimit = 10

	// Show this many evidence clusters in the text report. This only affects
	// display, not analysis.
	topClusterLimit = 10
)
