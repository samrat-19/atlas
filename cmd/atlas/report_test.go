package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atlas/internal/collector"
)

func TestPrintCountMapSortsLabels(t *testing.T) {
	var out strings.Builder

	printCountMap("Counts:", map[string]int{
		"zeta":  1,
		"alpha": 2,
		"beta":  3,
	}, &out)

	want := "Counts:\n- alpha: 2\n- beta: 3\n- zeta: 1\n"
	if out.String() != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", out.String(), want)
	}
}

// TestWriteReportsFailsWhenOutputDirIsFile verifies that writeReports returns
// a non-nil error when the output directory cannot be created. This guards
// against the previous behaviour of silently discarding write failures.
func TestWriteReportsFailsWhenOutputDirIsFile(t *testing.T) {
	// Change into a temp dir so writeReports tries to create "output" there.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tmp := t.TempDir()

	// Register the chdir-back cleanup after t.TempDir(), not before. Cleanups
	// run in LIFO order, so this must be registered second to run first —
	// otherwise t.TempDir()'s own RemoveAll cleanup runs while tmp is still
	// the process's working directory. Windows holds a lock on a directory
	// that is the current working directory, so RemoveAll fails with "the
	// process cannot access the file because it is being used by another
	// process" before we ever get a chance to chdir away.
	t.Cleanup(func() { _ = os.Chdir(orig) })

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// Place a regular file at the path writeReports will try to use as a
	// directory. MkdirAll will fail because you cannot mkdir over a file.
	if err := os.WriteFile(filepath.Join(tmp, "output"), []byte(""), 0644); err != nil {
		t.Fatalf("write output file: %v", err)
	}

	err = writeReports(collector.Result{}, "report text")
	if err == nil {
		t.Fatal("writeReports returned nil; expected an error when output dir cannot be created")
	}
}

func TestPrintUnrecognizedClustersShowsNoneWhenEmpty(t *testing.T) {
	var out strings.Builder
	printUnrecognizedClusters(collector.UnrecognizedSummary{}, &out)

	want := "Unrecognized Extension Clusters: none\n"
	if out.String() != want {
		t.Fatalf("unexpected output:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestPrintUnrecognizedClustersShowsClusterDetails(t *testing.T) {
	summary := collector.UnrecognizedSummary{
		TotalUnrecognizedDirectories: 3,
		TotalUnrecognizedFiles:       900,
		Clusters: []collector.UnrecognizedExtensionCluster{
			{Extension: ".bzl", DirectoryCount: 2, TotalFiles: 600, ExamplePaths: []string{"a/b", "c/d"}},
		},
	}

	var out strings.Builder
	printUnrecognizedClusters(summary, &out)
	text := out.String()

	for _, want := range []string{
		"Total unrecognized directories: 3",
		"Total unrecognized files: 900",
		"- .bzl (2 directories, 600 files)",
		"  - a/b",
		"  - c/d",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestPrintMajorModulesSortsWithoutMutatingSummary(t *testing.T) {
	summary := collector.CompressedModuleSummary{
		TotalCandidates:    3,
		RetainedCandidates: 3,
		CompressionRatio:   1,
		Modules: []collector.ModuleCandidate{
			{Path: "b", Score: 10, FileCount: 1, EvidenceCount: 1},
			{Path: "a", Score: 10, FileCount: 1, EvidenceCount: 1},
			{Path: "c", Score: 20, FileCount: 1, EvidenceCount: 1},
		},
	}

	var out strings.Builder
	printMajorModules(summary, &out)

	if summary.Modules[0].Path != "b" || summary.Modules[1].Path != "a" || summary.Modules[2].Path != "c" {
		t.Fatalf("printMajorModules mutated module order: %#v", summary.Modules)
	}

	text := out.String()
	cIndex := strings.Index(text, "- c\n")
	aIndex := strings.Index(text, "- a\n")
	bIndex := strings.Index(text, "- b\n")
	if cIndex == -1 || aIndex == -1 || bIndex == -1 {
		t.Fatalf("missing module output:\n%s", text)
	}
	if !(cIndex < aIndex && aIndex < bIndex) {
		t.Fatalf("modules not in deterministic order:\n%s", text)
	}
}
