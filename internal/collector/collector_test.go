package collector

import (
	"encoding/json"
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

// TestPruneSkipsDirectoryAndRecordsIt verifies the core pruning contract:
// a pruned directory's files must not appear in TotalFiles, the directory
// must appear in PrunedPaths, and no evidence from inside it must be collected.
func TestPruneSkipsDirectoryAndRecordsIt(t *testing.T) {
	dir := t.TempDir()

	// node_modules is in the prune list. Put a package.json inside it — if
	// pruning is broken, that file would appear as evidence and inflate counts.
	nm := filepath.Join(dir, "node_modules", "some-lib")
	if err := os.MkdirAll(nm, 0755); err != nil {
		t.Fatalf("mkdir node_modules/some-lib: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nm, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	// A real source file at the root so TotalFiles is not trivially zero.
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte(""), 0644); err != nil {
		t.Fatalf("write index.js: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	// Only index.js should be counted; nothing inside node_modules.
	if res.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1 (node_modules must not be counted)", res.TotalFiles)
	}

	// The evidence inside node_modules must not appear.
	if res.TopologySummary.TotalEvidenceItems != 0 {
		t.Fatalf("TotalEvidenceItems = %d, want 0 (node_modules evidence must be excluded)",
			res.TopologySummary.TotalEvidenceItems)
	}

	// node_modules must be recorded in PrunedPaths.
	if len(res.PrunedPaths) != 1 {
		t.Fatalf("PrunedPaths = %v, want exactly one entry", res.PrunedPaths)
	}
	if res.PrunedPaths[0].RelativePath != "node_modules" {
		t.Fatalf("PrunedPaths[0].RelativePath = %q, want %q", res.PrunedPaths[0].RelativePath, "node_modules")
	}
	if res.PrunedPaths[0].Policy != "installed-dependencies" {
		t.Fatalf("PrunedPaths[0].Policy = %q, want %q", res.PrunedPaths[0].Policy, "installed-dependencies")
	}
}

// TestPruneGitDirectory verifies that .git is excluded. This matters because
// .git contains paths like ".git/description" that would otherwise produce
// false evidence matches.
func TestPruneGitDirectory(t *testing.T) {
	dir := t.TempDir()

	// Simulate a minimal .git layout.
	if err := os.MkdirAll(filepath.Join(dir, ".git", "refs", "heads"), 0755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "description"), []byte(""), 0644); err != nil {
		t.Fatalf("write .git/description: %v", err)
	}
	// A real source file so the scan is not empty.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(""), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if res.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1 (.git files must not be counted)", res.TotalFiles)
	}

	found := false
	for _, p := range res.PrunedPaths {
		if p.RelativePath == ".git" {
			found = true
			if p.Policy != "version-control" {
				t.Fatalf(".git policy = %q, want %q", p.Policy, "version-control")
			}
		}
	}
	if !found {
		t.Fatalf(".git not found in PrunedPaths: %v", res.PrunedPaths)
	}
}

// TestPruneMultipleDirectories confirms that several prune-eligible sibling
// directories are all recorded when they appear in the same scan.
func TestPruneMultipleDirectories(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{".git", "node_modules", "bazel-out", "__pycache__"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "file.txt"), []byte(""), 0644); err != nil {
			t.Fatalf("write %s/file.txt: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte(""), 0644); err != nil {
		t.Fatalf("write source.go: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if res.TotalFiles != 1 {
		t.Fatalf("TotalFiles = %d, want 1 (all noise dirs excluded)", res.TotalFiles)
	}
	if len(res.PrunedPaths) != 4 {
		t.Fatalf("PrunedPaths count = %d, want 4", len(res.PrunedPaths))
	}
}

// TestCollectIsDeterministic scans the same fixture twice and compares the
// JSON-serialised Result byte-for-byte. Any nondeterminism — unsorted maps,
// missing sort tie-breakers, or traversal-order dependence — will surface here.
func TestCollectIsDeterministic(t *testing.T) {
	dir := t.TempDir()

	// Build a fixture with evidence, prunable dirs, and nested structure so
	// every code path (extensions, clusters, modules, hierarchy) is exercised.
	paths := []string{
		"src/main.go",
		"src/util.go",
		"src/go.mod",
		"web/package.json",
		"web/index.ts",
		"web/node_modules/dep/package.json", // must be pruned
		".git/config",                        // must be pruned
		"Dockerfile",
	}
	for _, p := range paths {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
		if err := os.WriteFile(full, []byte(""), 0644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	marshal := func(r Result) []byte {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		return b
	}

	first, err := Collect(dir)
	if err != nil {
		t.Fatalf("first Collect: %v", err)
	}
	second, err := Collect(dir)
	if err != nil {
		t.Fatalf("second Collect: %v", err)
	}

	a, b := marshal(first), marshal(second)
	if string(a) != string(b) {
		t.Fatalf("Collect is nondeterministic:\nfirst:  %s\nsecond: %s", a, b)
	}
}

// TestSchemaVersionPresent confirms that every Result carries a schema version.
// Consumers rely on this field to detect breaking changes.
func TestSchemaVersionPresent(t *testing.T) {
	dir := t.TempDir()
	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if res.SchemaVersion == "" {
		t.Fatalf("SchemaVersion is empty; every Result must carry a version")
	}
}

// TestCollectEvidenceConfidenceAtRootIsFull verifies that evidence found at
// an ordinary, non-noise-adjacent path keeps the rule's full confidence.
func TestCollectEvidenceConfidenceAtRootIsFull(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(res.Evidence) != 1 {
		t.Fatalf("expected 1 evidence item, got %d", len(res.Evidence))
	}
	if res.Evidence[0].Confidence != 1.0 {
		t.Fatalf("Confidence = %v, want 1.0", res.Evidence[0].Confidence)
	}
}

// TestCollectEvidenceConfidenceDiscountedUnderTestdata verifies that evidence
// found beneath a noise-adjacent directory (testdata, fixtures, examples,
// mocks) is recorded with reduced confidence rather than excluded outright.
func TestCollectEvidenceConfidenceDiscountedUnderTestdata(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "internal", "testdata", "sample")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "go.mod"), []byte("module x"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	res, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(res.Evidence) != 1 {
		t.Fatalf("expected 1 evidence item, got %d", len(res.Evidence))
	}
	if res.Evidence[0].Confidence != noiseAdjacentConfidenceMultiplier {
		t.Fatalf("Confidence = %v, want %v", res.Evidence[0].Confidence, noiseAdjacentConfidenceMultiplier)
	}
}

// TestMatchEvidenceSuffixRuleCarriesConfidence verifies that path-suffix
// rules (e.g. ".github/workflows") also carry confidence through
// MatchEvidence, not just plain-filename rules.
func TestMatchEvidenceSuffixRuleCarriesConfidence(t *testing.T) {
	_, category, confidence, ok := MatchEvidence("workflows", filepath.FromSlash(".github/workflows"))
	if !ok {
		t.Fatalf("expected a match for .github/workflows")
	}
	if category != "ci/cd" {
		t.Fatalf("category = %q, want ci/cd", category)
	}
	if confidence != 1.0 {
		t.Fatalf("confidence = %v, want 1.0", confidence)
	}
}

// TestMatchEvidenceNoMatchReturnsZeroConfidence verifies the no-match case
// still returns a zero confidence alongside ok=false, rather than a stale or
// undefined value.
func TestMatchEvidenceNoMatchReturnsZeroConfidence(t *testing.T) {
	_, _, confidence, ok := MatchEvidence("not-evidence.txt", "not-evidence.txt")
	if ok {
		t.Fatalf("expected no match for not-evidence.txt")
	}
	if confidence != 0 {
		t.Fatalf("confidence = %v, want 0 on no match", confidence)
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
