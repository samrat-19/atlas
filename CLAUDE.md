# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./cmd/atlas/

# Run (analyzes a directory, defaults to cwd)
go run ./cmd/atlas/ [path]

# Test all
go test ./...

# Test a single package
go test ./internal/collector/
go test ./internal/model/

# Run a specific test
go test ./internal/collector/ -run TestCollectCountsFiles

# Vet
go vet ./...

# Battery tests (requires large repos and ATLAS_BATTERY=1)
ATLAS_BATTERY=1 go test ./tests/battery/
```

Output reports (text + JSON) are written to `output/` relative to wherever the binary is invoked.

## Architecture

`atlas` is a Go CLI tool that walks a directory tree and produces a structured report about its topology — file distribution, build/infra "evidence" files, module candidates, and a repository hierarchy.

No external dependencies; pure stdlib.

### Packages

Two packages, deliberately not more — see `docs/phase-2-plan.md`'s package-boundary discussion for why a larger split (one package per pipeline stage) was rejected as premature for this codebase's size.

- **`internal/model`** — shared vocabulary, zero dependencies on anything else in this repo. `Result` and all its summary types, `ModuleCandidate`, `RegionNode`, `Role` (the type and its constants), `EvidenceRule`, `HeuristicProfile` + `DefaultHeuristics`, and the static directory-name pattern data (`VendoredPathSegments`, `GeneratedPathSegments`, `BuildOutputPathSegments`, `NoiseAdjacentPathSegments`, `PathContainsSegment()`). Pure data — no behavior beyond the one generic pattern-matching helper.
- **`internal/collector`** — all the logic. Imports `model`; `model` never imports it back. Walks the filesystem, matches evidence, scores and compresses candidates, classifies roles, builds the hierarchy.

The pattern data lives in `model`, not paired with `EvidenceRegistry` in `collector`, because it's consumed by two otherwise-independent concerns — evidence confidence discounting and role classification — that shouldn't depend on each other. Putting it in the shared base lets both import it without either importing the other.

### Data flow

```
cmd/atlas/main.go
  └─ collector.Collect(root)          // single FS walk, returns model.Result
       ├─ registry.MatchEvidence()    // classifies files as evidence (model.EvidenceRule)
       ├─ summaries.go                // updates model.TopologySummary, ClusterSummary, CensusSummary, ExtensionSummary
       ├─ directory_stats.go          // per-directory dirStat accumulation (collector-internal, not in model)
       ├─ modules.go                  // buildModuleSummary() — candidate selection + scoring (model.ModuleCandidate)
       ├─ role.go                     // classifyRole() — called from collect.go after buildModuleSummary returns,
       │                              //   not from modules.go itself; scoring and classification don't call each other
       ├─ module_scoring.go           // scoreModules() — compression scoring with overlap penalties
       ├─ module_compression.go       // compressModules() — prunes redundant parent-child candidates
       ├─ hierarchy.go                // buildHierarchy() — model.RegionNode tree from retained modules
       │    └─ hierarchy_aggregation.go  // aggregateRegion/subtree, sortRegionTree, copyRegionNode
       └─ unrecognized.go             // buildUnrecognizedSummary() — evidence-less candidates grouped by extension
  └─ renderReport(result)             // report.go: formats all summaries to a string
       ├─ report_structure.go         // printRepositoryHierarchy, printMajorModules, printUnrecognizedClusters, printTopClusters, printTopDirectories
       ├─ report_extensions.go        // printTopExtensions
       └─ report_limits.go            // display constants (topExtensionLimit, topClusterLimit, etc.)
  └─ writeReports(result, report)     // output.go: writes text + JSON to output/
```

### Key concepts

- **Evidence**: Files matching `EvidenceRegistry`, a `map[string]model.EvidenceRule` (build files, package managers, CI/CD configs, containers/IaC). Each match produces a `model.EvidenceItem` with a category and a confidence (0–1) — confidence starts at the rule's intrinsic value and is discounted when the match sits under a noise-adjacent directory (`test`, `fixtures`, `examples`, `mocks`, etc. — see `pathContextMultiplier` in `registry.go`, reading `model.NoiseAdjacentPathSegments`).
- **Cluster**: Top-level directory under root; all evidence and file counts are rolled up to this level for the `ClusterSummary`.
- **dirStat**: Internal per-directory accumulator (`directory_stats.go`), collector-only — never moved to `model` since nothing outside this package needs it. Tracks `FileCount`, `EvidenceCount`, `EvidenceConfidenceSum`, extension counts, and evidence breakdowns per category/filename. Used downstream for module candidate construction.
- **Module candidate**: A directory with evidence, sufficient file density, or large file count. Selected in `modules.go`; qualified by constants in `model.HeuristicProfile`. Carries a legacy `Score` (`EvidenceCount*100 + FileCount`, kept for backward-compatible sorting) plus five named, explainable 0–1 dimensions introduced in Phase 2 D3 — `EvidenceStrength`, `NoiseProbability`, `StructuralProminence`, `NoveltyVsParent`, `BoundaryConfidence` — computed from the evidence confidence above and from parent/subtree context. See the doc comment on `ModuleCandidate` in `internal/model/types.go`.
- **Compressed modules**: `compressModules()` (`module_compression.go`) prunes parent–child pairs using extension Jaccard overlap and category overlap. Children that are highly similar to their parent (`highOverlapThreshold=0.9`) receive a score penalty; final retention is based on relative score and novelty thresholds. `isStrongComparedToParent` clamps a non-positive parent score to zero before applying the ratio — without this, a deeply redundant parent (negative score) makes the bar easier, not harder, to clear.
- **Hierarchy**: `buildHierarchy()` (`hierarchy.go`) converts retained modules into a `model.RegionNode` tree (Regions → subsystems → components). Aggregation rolls child file/evidence counts up in `hierarchy_aggregation.go`; the final tree is sorted by score descending. `RegionNode` does not yet carry the five D3 dimensions or `Role` — aggregating non-additive 0–1 scores (or a label) across a subtree has no settled answer yet.
- **Unrecognized clusters**: `buildUnrecognizedSummary()` (`unrecognized.go`) groups module candidates that qualified purely by size (zero evidence) by shared dominant extension. A diagnostic for finding evidence-registry/role gaps from real repository structure — not part of scoring, classification, or retention.
- **Structural role**: `classifyRole()` (`role.go`, Phase 2 D4a) labels each module candidate `model.RoleFirstParty`, `RoleVendored`, `RoleGenerated`, `RoleTestFixture`, `RoleBuildOutput`, or `RoleAmbiguous`, via a fixed, ordered list of directory-name pattern checks (`model`'s pattern data) followed by an evidence-strength fallback. Called from `collect.go`'s orchestration, after `buildModuleSummary` returns — not from inside `modules.go` — so the scoring code never imports classification. A label only — it does not currently affect `Score`, retention, or compression (that's the separate, deliberately deferred D4b decision).

### Package layout

| Path | Role |
|---|---|
| `cmd/atlas/main.go` | Entry point; arg parsing, orchestrates collect → render → write |
| `cmd/atlas/report.go` | `renderReport()`, `printEvidenceSummary()`, `printCountMap()` |
| `cmd/atlas/report_structure.go` | Hierarchy, major modules, clusters, top-directory printers |
| `cmd/atlas/report_extensions.go` | Extension count formatting |
| `cmd/atlas/report_limits.go` | Display constants (how many items to show in each section) |
| `cmd/atlas/output.go` | `writeReports()` — writes timestamped `.txt` and `.json` to `output/` |
| `cmd/atlas/report_test.go` | Unit tests for report formatting |
| `internal/model/types.go` | `Result` and every summary type, `ModuleCandidate`, `RegionNode`, `EvidenceRule` |
| `internal/model/role.go` | `Role` type + constants (`RoleFirstParty`, `RoleVendored`, etc.) — no logic |
| `internal/model/patterns.go` | Directory-name pattern data (`VendoredPathSegments`, etc.) + `PathContainsSegment()` |
| `internal/model/heuristics.go` | `HeuristicProfile` and sub-configs, `DefaultHeuristics`, `CurrentSchemaVersion` |
| `internal/collector/registry.go` | `EvidenceRegistry`, `MatchEvidence()`, `SetRegistry()`, `pathContextMultiplier()` |
| `internal/collector/collect.go` | `Collect()` — FS walk, orchestration, calls `classifyRole` after scoring |
| `internal/collector/summaries.go` | `updateWith*`-style functions for `model.TopologySummary`, `ClusterSummary`, `CensusSummary`, `ExtensionSummary` |
| `internal/collector/directory_stats.go` | `dirStat` type (collector-internal) and `directoryStatsFor()` |
| `internal/collector/modules.go` | `buildModuleSummary()`, candidate selection, dominant extension logic — no role classification |
| `internal/collector/module_scoring.go` | `scoreModules()`, Jaccard overlap helpers |
| `internal/collector/module_compression.go` | `compressModules()`, parent resolution, retention logic |
| `internal/collector/hierarchy.go` | `buildHierarchy()`, parent resolution, node attachment |
| `internal/collector/hierarchy_aggregation.go` | `aggregateRegion/Subtree()`, `sortRegionTree()`, `copyRegionNode()` |
| `internal/collector/unrecognized.go` | `buildUnrecognizedSummary()` — groups evidence-less candidates by dominant extension |
| `internal/collector/role.go` | `classifyRole()` — ordered, deterministic structural-role labeling logic |
| `internal/collector/collector_test.go` | Unit tests using `t.TempDir()` temp fixtures |
| `tests/battery/battery_test.go` | Battery tests against real large repos (selenium, tensorflow, vscode); opt-in via `ATLAS_BATTERY=1` |

### Extending evidence categories

Add entries to `defaultEvidenceRegistry()` in `registry.go` as `model.EvidenceRule{Category, Confidence}` values. Keys are either plain filenames (matched by `filenameIndex`) or relative-path suffixes containing `/` (matched by `suffixIndex`). Call `SetRegistry(nil)` in tests to restore defaults.

### Tuning heuristics

Every tunable number lives in a single `model.HeuristicProfile` value (`internal/model/heuristics.go`), grouped by which pipeline stage reads it. `model.DefaultHeuristics` is the only profile that exists today; it is threaded as an explicit parameter through `MatchEvidence`, `buildModuleSummary`, `scoreModules`, and `compressModules` rather than read from package constants inside those functions — so a future alternate profile only requires constructing a new `HeuristicProfile` and passing it to `Collect`. Key fields:

| Field | Default | Effect |
|---|---|---|
| `EvidenceConfidence.NoiseAdjacentConfidenceMultiplier` | 0.5 | Confidence discount for evidence found under test/fixture/example-style directories |
| `CandidateSelection.LargeDirectoryFileThreshold` | 200 | File count to qualify a dir as a candidate without evidence |
| `CandidateSelection.EvidenceDensityThreshold` | 0.05 | Min evidence/file ratio to qualify a dense dir |
| `Scoring.ModuleEvidenceWeight` | 100 | Score weight per evidence file in module selection |
| `Compression.CompressionEvidenceWeight` | 200 | Score weight per evidence file during compression |
| `Compression.HighOverlapThreshold` | 0.9 | Extension + category similarity above which a child is penalized |
| `Compression.ChildScoreRetentionRatio` | 0.6 | Min child/parent score ratio to retain a child |
| `Compression.NoveltyRetentionDelta` | 0.2 | Min dissimilarity (1 - overlap) to retain a novel child |
