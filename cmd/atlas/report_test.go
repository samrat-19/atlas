package main

import (
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
