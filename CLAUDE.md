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

### Data flow

```
cmd/atlas/main.go
  └─ collector.Collect(root)          // single FS walk, returns Result
       ├─ registry.MatchEvidence()    // classifies files as evidence
       ├─ summaries.go                // updates TopologySummary, ClusterSummary, CensusSummary, ExtensionSummary
       ├─ directory_stats.go          // per-directory dirStat accumulation
       ├─ modules.go                  // buildModuleSummary() — candidate selection + scoring
       ├─ module_scoring.go           // scoreModules() — compression scoring with overlap penalties
       ├─ module_compression.go       // compressModules() — prunes redundant parent-child candidates
       └─ hierarchy.go                // buildHierarchy() — RegionNode tree from retained modules
            └─ hierarchy_aggregation.go  // aggregateRegion/subtree, sortRegionTree, copyRegionNode
  └─ renderReport(result)             // report.go: formats all summaries to a string
       ├─ report_structure.go         // printRepositoryHierarchy, printMajorModules, printTopClusters, printTopDirectories
       ├─ report_extensions.go        // printTopExtensions
       └─ report_limits.go            // display constants (topExtensionLimit, topClusterLimit, etc.)
  └─ writeReports(result, report)     // output.go: writes text + JSON to output/
```

### Key concepts

- **Evidence**: Files matching `EvidenceRegistry`, a `map[string]EvidenceRule` (build files, package managers, CI/CD configs, containers/IaC). Each match produces an `EvidenceItem` with a category and a confidence (0–1) — confidence starts at the rule's intrinsic value and is discounted when the match sits under a noise-adjacent directory (`test`, `fixtures`, `examples`, `mocks`, etc. — see `pathContextMultiplier` in `registry.go`). Confidence is not yet consumed by scoring (Phase 2 in progress).
- **Cluster**: Top-level directory under root; all evidence and file counts are rolled up to this level for the `ClusterSummary`.
- **dirStat**: Internal per-directory accumulator (`directory_stats.go`). Tracks `FileCount`, `EvidenceCount`, extension counts, and evidence breakdowns per category/filename. Used downstream for module candidate construction.
- **Module candidate**: A directory with evidence, sufficient file density, or large file count. Selected in `modules.go`; qualified by constants in `heuristics.go`. Scored by `EvidenceCount*100 + FileCount`.
- **Compressed modules**: `compressModules()` (`module_compression.go`) prunes parent–child pairs using extension Jaccard overlap and category overlap. Children that are highly similar to their parent (`highOverlapThreshold=0.9`) receive a score penalty; final retention is based on relative score and novelty thresholds.
- **Hierarchy**: `buildHierarchy()` (`hierarchy.go`) converts retained modules into a `RegionNode` tree (Regions → subsystems → components). Aggregation rolls child file/evidence counts up in `hierarchy_aggregation.go`; the final tree is sorted by score descending.

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
| `internal/collector/types.go` | All exported types |
| `internal/collector/registry.go` | `EvidenceRegistry`, `MatchEvidence()`, `SetRegistry()` |
| `internal/collector/collect.go` | `Collect()` — FS walk and result assembly |
| `internal/collector/summaries.go` | `updateWith*` methods on TopologySummary, ClusterSummary, CensusSummary, ExtensionSummary |
| `internal/collector/directory_stats.go` | `dirStat` type and `directoryStatsFor()` |
| `internal/collector/modules.go` | `buildModuleSummary()`, candidate selection, dominant extension logic |
| `internal/collector/module_scoring.go` | `scoreModules()`, Jaccard overlap helpers |
| `internal/collector/module_compression.go` | `compressModules()`, parent resolution, retention logic |
| `internal/collector/heuristics.go` | All tuning constants with explanatory comments |
| `internal/collector/hierarchy.go` | `buildHierarchy()`, parent resolution, node attachment |
| `internal/collector/hierarchy_aggregation.go` | `aggregateRegion/Subtree()`, `sortRegionTree()`, `copyRegionNode()` |
| `internal/collector/collector_test.go` | Unit tests using `t.TempDir()` temp fixtures |
| `tests/battery/battery_test.go` | Battery tests against real large repos (selenium, tensorflow, vscode); opt-in via `ATLAS_BATTERY=1` |

### Extending evidence categories

Add entries to `defaultEvidenceRegistry()` in `registry.go` as `EvidenceRule{Category, Confidence}` values. Keys are either plain filenames (matched by `filenameIndex`) or relative-path suffixes containing `/` (matched by `suffixIndex`). Call `SetRegistry(nil)` in tests to restore defaults.

### Tuning heuristics

All scoring constants live in `heuristics.go` with inline explanations. Key ones:

| Constant | Default | Effect |
|---|---|---|
| `largeDirectoryFileThreshold` | 200 | File count to qualify a dir as a candidate without evidence |
| `evidenceDensityThreshold` | 0.05 | Min evidence/file ratio to qualify a dense dir |
| `moduleEvidenceWeight` | 100 | Score weight per evidence file in module selection |
| `compressionEvidenceWeight` | 200 | Score weight per evidence file during compression |
| `highOverlapThreshold` | 0.9 | Extension + category similarity above which a child is penalized |
| `childScoreRetentionRatio` | 0.6 | Min child/parent score ratio to retain a child |
| `noveltyRetentionDelta` | 0.2 | Min dissimilarity (1 - overlap) to retain a novel child |
