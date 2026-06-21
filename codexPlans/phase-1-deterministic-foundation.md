# Phase 1 Plan: Deterministic Foundation

## Document Status

- Phase: 1 of 3
- Status: In progress
- Created: 2026-06-15
- Updated: 2026-06-21
- Project: Atlas

## 1. Purpose

Before Atlas can classify importance or orient engineers reliably, its
observations must be correct, stable, and reproducible.

Phase 1 establishes that foundation through four focused deliverables. It does
not attempt to redesign the evidence engine, restructure packages, or introduce
configuration frameworks. Those belong to Phase 2.

## 2. Deliverables

### D1: Prune noise by default

Atlas currently traverses `.git`, `node_modules`, `bazel-out`, and similar
directories. These contaminate file counts, evidence counts, and module
candidates.

Phase 1 must:

- Skip traversal of high-confidence operational directories by default.
- Record each pruned path in `Result` so the snapshot is honest about what was
  skipped.
- Not silently inflate or deflate statistics with noise.

Initial prune list (high-confidence only):

| Directory | Reason |
| --- | --- |
| `.git` | Version-control metadata |
| `node_modules` | Installed dependencies |
| `bazel-out` | Bazel build output |
| `.terraform` | Terraform plugin cache |
| `__pycache__` | Python bytecode cache |
| `.cache` | Generic cache |
| `dist` | Common build output (JS/TS ecosystems) |
| `build` | **Not pruned** — too ambiguous; a Go or Bazel `build/` can be source |
| `vendor` | **Not pruned** by default — ecosystem-dependent; recorded as classified |
| `third_party` | **Not pruned** — may contain important source in large repos |

For each pruned directory, `Result` should record:
- relative path
- the policy name that triggered the prune
- file count if known (unknown if pruning prevents traversal)

### D2: Fix remaining nondeterminism

The battery tests already enforce output stability, but a few sources remain:

- Report code sorts maps at render time, not at collection time. A sort with no
  complete tie-breaker is nondeterministic when values are equal.
- `printMajorModules` was fixed (does not mutate the snapshot). Verify all other
  renderers follow the same rule.

Phase 1 must:

- Ensure every slice that feeds the report has a fully specified sort order
  (primary field, then secondary, then path as final tie-breaker).
- Ensure no renderer mutates the snapshot.
- Run the battery tests clean after each change.

### D3: Fix output error handling

Report write errors are currently silently discarded in `output.go`. This means
a failed write looks like success.

Phase 1 must:

- Return errors from `writeReports`.
- Exit with a nonzero code when output cannot be written.
- Distinguish scan failure (exit 2) from output failure (exit 3).

### D4: Add schema version to JSON output

The current JSON reflects the Go struct layout with no version marker. Consumers
have no way to detect when the shape changes.

Phase 1 must:

- Add a `schemaVersion` string field to the JSON output.
- Set it to `"1"` for Phase 1 output.
- No new package or serialization layer is needed — a single field in `Result`
  is sufficient.

Canonical JSON must also exclude timestamps and machine-specific absolute root
paths, or move them to a clearly labeled runtime envelope.

## 3. Explicitly Out of Scope

- Typed evidence rules with stable IDs (`EvidenceRule` structs)
- New packages (`internal/evidence/`, `internal/policy/`, `internal/snapshot/`)
- `ScanConfiguration` model
- Git-aware traversal
- Symlink and junction policy
- CLI redesign (`atlas scan <path>`)
- Importance ranking or structural-role classification
- Source-code parsing
- AI-generated summaries

These are Phase 2 or later.

## 4. Test Repositories

| Repository | Structural challenge |
| --- | --- |
| Selenium | Polyglot monorepo, Bazel, multiple language bindings |
| TensorFlow | Very large Bazel repo, generated areas, third-party content |
| VS Code | TypeScript/Electron, Node ecosystem noise, extensions |

Battery tests for all three are already in `tests/battery/`. After each
deliverable, the battery fixtures must be updated if output changes are expected,
and must pass clean otherwise.

## 5. Testing Strategy

- **Unit tests** for prune logic: a pruned directory is not traversed, its path
  appears in `Result`, its files do not appear in `TotalFiles`.
- **Fixture tests** for nondeterminism: scan the same fixture twice, compare
  bytes.
- **Battery tests** for regression: all three repositories must produce stable
  output after noise is removed (fixtures will need updating once D1 lands).
- **Exit code tests** for D3: simulate a write failure and assert the exit code.

## 6. Implementation Order

1. D1 — prune noise. Update battery fixtures to reflect cleaner output.
2. D2 — verify and fix any remaining nondeterminism exposed by D1.
3. D3 — fix output error handling.
4. D4 — add schema version field.

Each step leaves `go test ./...` and `go vet ./...` passing.

## 7. Exit Criteria

Phase 1 is complete when:

- `.git` is not traversed or counted.
- `node_modules` and `bazel-out` are not traversed and are not promoted as module candidates.
- Every pruned path appears in `Result`.
- Battery tests pass cleanly for all three repositories.
- All sort orders have complete tie-breakers.
- No renderer mutates the snapshot.
- Output write failures produce a nonzero exit code.
- JSON output contains `schemaVersion: "1"`.
- `go test ./...` and `go vet ./...` pass.

## 8. Phase 2 Handoff

Phase 2 receives:

- Clean observations (noise excluded and recorded).
- Stable, reproducible output.
- Battery baselines for all three repositories.
- A list of known classification inaccuracies (false-positive module candidates,
  missing regions) identified during D1 evaluation.

Phase 2 will focus on evidence rule redesign, structural-role classification,
multidimensional scoring, and confidence signals.
