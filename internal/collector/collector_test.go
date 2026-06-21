package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectCountsFiles(t *testing.T) {
	dir := t.TempDir()

	// create two files in different directories
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatalf("write b.txt: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if res.TotalFiles != 2 {
		t.Fatalf("expected 2 files, got %d", res.TotalFiles)
	}
}

func TestCollectEmptyDir(t *testing.T) {
	dir := t.TempDir()

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if res.TotalFiles != 0 {
		t.Fatalf("expected 0 files, got %d", res.TotalFiles)
	}
}

func TestCollectInvalidPath(t *testing.T) {
	// point to a non-existent path
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	_, err := Collect(dir)
	if err == nil {
		t.Fatalf("expected error for non-existent path, got nil")
	}
}

func TestCollectEvidenceDefaultRegistry(t *testing.T) {
	dir := t.TempDir()

	// create evidence files/dirs
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch"), 0644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	// create .github/workflows/ci.yml
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "ci.yml"), []byte("name: CI"), 0644); err != nil {
		t.Fatalf("write ci.yml: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	// Expect 3 files
	if res.TotalFiles != 3 {
		t.Fatalf("expected 3 files, got %d", res.TotalFiles)
	}

	if res.TopologySummary.TotalEvidenceItems != 3 {
		t.Fatalf("expected total evidence items 3, got %d", res.TopologySummary.TotalEvidenceItems)
	}
	if res.TopologySummary.RootEvidenceCount != 2 {
		t.Fatalf("expected root-level evidence count 2, got %d", res.TopologySummary.RootEvidenceCount)
	}
	if res.TopologySummary.NestedEvidenceCount != 1 {
		t.Fatalf("expected nested evidence count 1, got %d", res.TopologySummary.NestedEvidenceCount)
	}

	if res.TopologySummary.EvidenceCountByCategory["package manager"] != 1 {
		t.Fatalf("expected 1 package manager evidence, got %d", res.TopologySummary.EvidenceCountByCategory["package manager"])
	}
	if res.TopologySummary.EvidenceCountByCategory["container"] != 1 {
		t.Fatalf("expected 1 container evidence, got %d", res.TopologySummary.EvidenceCountByCategory["container"])
	}
	if res.TopologySummary.EvidenceCountByCategory["ci/cd"] != 1 {
		t.Fatalf("expected 1 ci/cd evidence, got %d", res.TopologySummary.EvidenceCountByCategory["ci/cd"])
	}

	// Ensure evidence contains the three expected registry keys
	want := map[string]string{
		"package.json":      "package manager",
		"Dockerfile":        "container",
		".github/workflows": "ci/cd",
	}
	found := make(map[string]bool)
	for _, e := range res.Evidence {
		if cat, ok := want[e.Filename]; ok {
			if e.Category != cat {
				t.Fatalf("evidence %s had category %s, want %s", e.Filename, e.Category, cat)
			}
			if e.AbsolutePath == "" {
				t.Fatalf("evidence %s missing absolute path", e.Filename)
			}
			if e.RelativePath == "" {
				t.Fatalf("evidence %s missing relative path", e.Filename)
			}
			found[e.Filename] = true
		}
	}
	for k := range want {
		if !found[k] {
			t.Fatalf("expected evidence %s not found", k)
		}
	}
}

func TestCollectCensusSummary(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "java", "client"), 0755); err != nil {
		t.Fatalf("mkdir java/client: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "third_party"), 0755); err != nil {
		t.Fatalf("mkdir third_party: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "java", "client", "BUILD.bazel"), []byte(""), 0644); err != nil {
		t.Fatalf("write BUILD.bazel: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "java", "client", "Main.java"), []byte(""), 0644); err != nil {
		t.Fatalf("write Main.java: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "third_party", "README.md"), []byte(""), 0644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if res.CensusSummary.TotalDirectories != 3 {
		t.Fatalf("expected 3 directories, got %d", res.CensusSummary.TotalDirectories)
	}

	java, ok := res.CensusSummary.Directories["java"]
	if !ok {
		t.Fatalf("expected java directory census")
	}
	if java.TotalFiles != 2 {
		t.Fatalf("expected java total files 2, got %d", java.TotalFiles)
	}
	if java.EvidenceItemCount != 1 {
		t.Fatalf("expected java evidence 1, got %d", java.EvidenceItemCount)
	}

	third, ok := res.CensusSummary.Directories["third_party"]
	if !ok {
		t.Fatalf("expected third_party directory census")
	}
	if third.TotalFiles != 1 {
		t.Fatalf("expected third_party total files 1, got %d", third.TotalFiles)
	}
	if third.EvidenceItemCount != 0 {
		t.Fatalf("expected third_party evidence 0, got %d", third.EvidenceItemCount)
	}

	root, ok := res.CensusSummary.Directories["_root"]
	if !ok {
		t.Fatalf("expected _root directory census")
	}
	if root.TotalFiles != 1 {
		t.Fatalf("expected _root total files 1, got %d", root.TotalFiles)
	}
	if root.EvidenceItemCount != 1 {
		t.Fatalf("expected _root evidence 1, got %d", root.EvidenceItemCount)
	}
}

func TestBuildHierarchyFromRetainedModules(t *testing.T) {
	mods := []ModuleCandidate{
		{Path: "java", FileCount: 100, EvidenceCount: 5, Score: 1200},
		{Path: "java/src", FileCount: 40, EvidenceCount: 2, Score: 800},
		{Path: "java/src/org/openqa/selenium/grid", FileCount: 20, EvidenceCount: 1, Score: 300},
		{Path: "javascript", FileCount: 60, EvidenceCount: 3, Score: 900},
		{Path: "javascript/selenium-webdriver", FileCount: 25, EvidenceCount: 1, Score: 500},
	}

	hs := buildHierarchy(mods)
	t.Logf("hierarchy=%#v", hs)
	if hs.TotalRegions != 2 {
		t.Fatalf("expected 2 regions, got %d", hs.TotalRegions)
	}
	if hs.TotalSubsystems != 3 {
		t.Fatalf("expected 3 subsystems, got %d", hs.TotalSubsystems)
	}

	javaRegion := hs.Regions[0]
	if javaRegion.Path != "java" {
		t.Fatalf("expected first region java, got %s", javaRegion.Path)
	}
	if javaRegion.FileCount < 100 {
		t.Fatalf("expected java file count at least 100, got %d", javaRegion.FileCount)
	}
	if len(javaRegion.Children) != 1 {
		t.Fatalf("expected java to have 1 child, got %d", len(javaRegion.Children))
	}
	if javaRegion.Children[0].Path != "java/src" {
		t.Fatalf("expected java child java/src, got %s", javaRegion.Children[0].Path)
	}
	if len(javaRegion.Children[0].Children) != 1 {
		t.Fatalf("expected java/src to have 1 child, got %d", len(javaRegion.Children[0].Children))
	}
	if javaRegion.Children[0].Children[0].Path != "java/src/org/openqa/selenium/grid" {
		t.Fatalf("expected nested component path, got %s", javaRegion.Children[0].Children[0].Path)
	}
}

func TestBuildModuleSummaryUsesPathTieBreaker(t *testing.T) {
	stats := map[string]*dirStat{
		"z": {
			FileCount:          1,
			EvidenceCount:      1,
			Extensions:         map[string]int{".go": 1},
			EvidenceByCategory: map[string]int{"package manager": 1},
			EvidenceByFilename: map[string]int{"go.mod": 1},
		},
		"a": {
			FileCount:          1,
			EvidenceCount:      1,
			Extensions:         map[string]int{".go": 1},
			EvidenceByCategory: map[string]int{"package manager": 1},
			EvidenceByFilename: map[string]int{"go.mod": 1},
		},
	}

	summary := buildModuleSummary(stats)
	if len(summary.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(summary.Modules))
	}
	if summary.Modules[0].Path != "a" || summary.Modules[1].Path != "z" {
		t.Fatalf("modules not ordered by path tie-breaker: %#v", summary.Modules)
	}
}

func TestDominantExtensionsUsesExtensionTieBreaker(t *testing.T) {
	extensions := dominantExtensions(map[string]int{
		".z": 1,
		".a": 1,
		".m": 2,
	}, 3)

	want := []string{".m", ".a", ".z"}
	for i := range want {
		if extensions[i] != want[i] {
			t.Fatalf("dominantExtensions[%d] = %q, want %q; full=%#v", i, extensions[i], want[i], extensions)
		}
	}
}
