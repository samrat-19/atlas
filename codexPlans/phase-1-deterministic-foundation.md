# Phase 1 Plan: Deterministic Foundation

## Document Status

- Phase: 1 of 3
- Status: Planned, awaiting review
- Created: 2026-06-15
- Project: Atlas
- Implementation approval: Not yet granted
- Primary objective: Make Atlas structurally trustworthy before improving its intelligence

## 1. Purpose

Atlas is intended to orient engineers inside unfamiliar repositories. Before Atlas
can classify importance, recommend investigation roots, or provide context to AI
agents, its observations must be correct, stable, reproducible, and explainable.

Phase 1 establishes that foundation.

This phase does not attempt to make Atlas fully intelligent. It makes Atlas a
dependable measurement engine whose output can support later classification and
recommendation logic.

The central requirement is:

> Given the same repository content, Atlas configuration, and Atlas version, the
> canonical structural result must be identical.

The second requirement is:

> Every evidence match and exclusion decision must be traceable to a named rule.

The third requirement is:

> Operational noise must not silently distort the repository model.

## 2. Relationship to the Atlas Vision

Atlas ultimately needs to answer:

- What kind of repository is this?
- Which technologies and build systems are present?
- Which regions are likely important?
- What can probably be ignored?
- Where should an engineer or tool begin?

Phase 1 addresses the prerequisites for those questions:

- Accurate filesystem observations
- Correct evidence matching
- Deterministic ordering
- Explicit treatment of excluded and noisy regions
- A canonical machine-readable repository snapshot
- Repeatable evaluation against real repositories

Without this phase, later importance scores and recommendations would be built on
unstable or contaminated input.

## 3. Test Repositories

The initial evaluation corpus consists of:

| Repository | Path | Expected structural challenge |
| --- | --- | --- |
| Selenium | `D:\SampleTestProjects\selenium-trunk` | Polyglot monorepo, multiple bindings, Bazel, language-specific regions |
| TensorFlow | `D:\SampleTestProjects\tensorflow-master` | Very large Bazel repository, generated areas, third-party content, deep hierarchy |
| VS Code | `D:\SampleTestProjects\vscode-main` | TypeScript/Electron product, extensions, build tooling, Node ecosystem noise |

These repositories are evaluation inputs. Atlas must not modify them.

The three repositories are not sufficient to prove general accuracy. They are
sufficient to establish deterministic behavior, identify major correctness
failures, and define the first measurable baseline.

## 4. Phase 1 Outcomes

At the end of Phase 1, Atlas should provide:

1. A deterministic canonical repository snapshot.
2. Explicit evidence rules with unambiguous matching semantics.
3. Explicit traversal policies for ignored, excluded, and classified regions.
4. Stable text and JSON output derived from the same snapshot.
5. A repeatable evaluation harness for the three test repositories.
6. Baseline performance and structural metrics.
7. Tests that prevent regressions in determinism and evidence correctness.
8. Clear documentation of known limitations deferred to Phase 2.

## 5. Scope

### 5.1 Included

- Deterministic traversal semantics
- Deterministic result ordering
- Deterministic report rendering
- Stable path normalization
- Evidence rule redesign
- Exact handling of files and directories
- Noise and exclusion policy
- Canonical snapshot data model
- Scan metadata required for reproducibility
- Configuration identity and rule-set versioning
- Evaluation tooling
- Regression tests
- Runtime and memory baseline collection
- Documentation of all Phase 1 decisions

### 5.2 Explicitly Excluded

- Final importance ranking
- Final structural-role classification
- Repository-type classification
- Task-specific investigation profiles
- AI-generated summaries
- Source-code parsing
- Dependency resolution
- Vulnerability scanning
- SBOM generation
- Hosted service or dashboard
- Machine learning
- Final Detect execution recommendations

Phase 1 may introduce data fields that later phases will use, but it must not
pretend to solve those later problems.

## 6. Existing Behavior and Known Risks

The current implementation:

- Walks the complete directory tree with `filepath.WalkDir`.
- Counts every encountered file.
- Matches evidence through filename and path-suffix maps.
- Groups observations by top-level directory.
- Builds module candidates from evidence count, file count, and evidence density.
- Scores and compresses candidates heuristically.
- Builds a hierarchy from retained candidates.
- Writes text and JSON reports.

Known Phase 1 correctness concerns include:

1. `.git` and other operational directories are scanned.
2. Installed dependencies and generated output can dominate statistics.
3. Windows case-insensitive matching can produce ambiguous matches.
4. File rules and directory rules are represented in the same registry format.
5. Some directory rules containing trailing slashes cannot match correctly.
6. Path-suffix matching may match unintended path fragments.
7. Map iteration causes nondeterministic ordering.
8. Sort operations do not consistently define tie-breakers.
9. Report persistence errors are ignored.
10. Timestamped output cannot serve as canonical output.
11. Current JSON reflects Go struct layout rather than a deliberately versioned
    result contract.
12. Module inference is currently coupled to raw observations and may consume
    noise-contaminated data.

Phase 1 should correct foundational behavior while preserving the current
heuristic model as much as practical. Changes to heuristic conclusions caused by
corrected input are acceptable and expected.

## 7. Core Design Principles

### 7.1 Observations Before Inferences

Filesystem facts must be captured independently from conclusions.

Examples of observations:

- A file exists at `javascript/package.json`.
- A directory contains 1,240 files.
- A directory was excluded by a known dependency-directory rule.
- A manifest rule matched an exact filename.

Examples of inferences:

- The directory is a probable module.
- The repository is likely a monorepo.
- A region is probably generated.

Phase 1 must ensure observations are reliable. Later phases may replace inference
logic without changing how the repository was measured.

### 7.2 Canonical Data Before Presentation

Text and JSON reports must be projections of one canonical snapshot. Reporting
code must not recalculate structural conclusions.

### 7.3 Explicit Rules

Rules must declare:

- Stable rule identifier
- Rule type
- Match target
- Category
- Match semantics
- Case sensitivity
- Whether the rule applies to files, directories, or both
- Whether it affects traversal
- Human-readable rationale

### 7.4 No Silent Disappearance

Atlas may skip traversal of known noise, but the snapshot should record:

- The skipped path
- The policy or rule that caused the skip
- The broad classification
- Whether file counts are known or unknown

Atlas should distinguish:

- Not encountered
- Encountered and included
- Encountered and classified
- Encountered and traversal-pruned
- Encountered but inaccessible

### 7.5 Stable Identifiers

Rules, schema versions, and classifications need stable identifiers so that
results remain explainable as Atlas evolves.

## 8. Proposed Architecture

The architecture should remain compact. Phase 1 should introduce ownership
boundaries, not an elaborate framework.

Potential package responsibilities:

```text
cmd/atlas/
  CLI orchestration
  configuration loading
  report selection
  exit behavior

internal/collector/
  filesystem traversal
  raw observations
  aggregate statistics

internal/evidence/
  evidence rule definitions
  rule compilation
  match evaluation

internal/policy/
  traversal and exclusion rules
  path classification

internal/snapshot/
  canonical result model
  schema version
  canonical ordering
  serialization

internal/evaluation/
  optional evaluation helpers if they belong in the application

codexPlans/
  phase plans
  decisions
  concise evaluation summaries
```

The final package split will be decided during implementation after reviewing
dependency direction. It is acceptable to keep some responsibilities in the
existing `collector` package if extracting them would add ceremony without
improving ownership.

Expected dependency direction:

```text
CLI
  -> scan orchestration
      -> traversal policy
      -> evidence matcher
      -> collector
      -> canonical snapshot
  -> report renderers
```

Reporters should depend on the canonical snapshot, not on collector internals.

## 9. Canonical Snapshot

### 9.1 Proposed Top-Level Model

The exact Go representation will be finalized during implementation. The
conceptual model should include:

```text
RepositorySnapshot
  schemaVersion
  atlasVersion
  ruleSetVersion
  root
  scanConfiguration
  repositoryStatistics
  evidence
  regions
  exclusions
  inaccessiblePaths
  currentInferences
```

### 9.2 Metadata

Metadata should distinguish canonical and noncanonical values.

Canonical metadata:

- Schema version
- Atlas version or build identity
- Rule-set version
- Normalized root identity, if included
- Scan configuration
- Enabled policies

Noncanonical runtime metadata:

- Scan start time
- Scan duration
- Host platform
- Absolute machine-specific paths

Runtime metadata should either be omitted from canonical serialization or stored
in a separate envelope.

### 9.3 Paths

Snapshot paths should:

- Be repository-relative wherever possible.
- Use `/` as the canonical separator.
- Represent repository root consistently, likely `"."`.
- Avoid embedding machine-specific absolute paths in canonical output.
- Preserve actual filename casing.
- Define behavior for symlinks and junctions.

Absolute paths may be available in runtime APIs, but should not prevent two
copies of the same repository from producing equivalent canonical snapshots.

### 9.4 Ordering

Every repeated collection must have a defined order.

Suggested ordering:

- Paths: normalized path ascending
- Evidence: relative path, then rule ID
- Regions: ranking fields, then normalized path
- Exclusions: relative path, then rule ID
- Categories: stable identifier ascending
- Extensions: count descending, then extension ascending
- Report rankings: primary score descending, then secondary score descending,
  then normalized path ascending

No canonical slice should inherit Go map iteration order.

### 9.5 Schema Version

The snapshot should contain a schema version from its first formal version.

Example:

```json
{
  "schemaVersion": "1.0"
}
```

Schema changes should follow a documented compatibility policy:

- Additive optional field: minor schema revision
- Meaning change or field removal: major schema revision

Phase 1 does not need a general migration system, but it must avoid treating the
current accidental JSON shape as a permanent public contract.

## 10. Determinism Requirements

### 10.1 Definition

For Phase 1, deterministic behavior means:

> Repeated scans of unchanged repository content with the same Atlas binary and
> configuration produce semantically identical and canonically byte-identical
> snapshots.

Text output should also be byte-identical when runtime metadata is excluded.

### 10.2 Sources of Nondeterminism to Remove

- Go map iteration
- Unspecified tie ordering in `sort.Slice`
- Filesystem enumeration differences
- Absolute path differences
- Timestamps in canonical artifacts
- Platform-specific separators
- Platform-specific case normalization
- Mutable global registry state
- Nondeterministic error ordering
- Output mutation caused by sorting slices shared with the snapshot

### 10.3 Determinism Test Strategy

Unit tests:

- Canonical serialization sorts every collection.
- Equal scores use path tie-breakers.
- Report rendering does not mutate the snapshot.
- Canonical paths use `/`.
- Runtime metadata is excluded from canonical hashes.

Integration tests:

- Scan a fixture repeatedly and compare bytes.
- Scan equivalent fixture copies under different absolute roots.
- Run concurrent independent scans and compare results.
- Verify text and JSON are stable across repeated runs.

Repository evaluation:

- Scan each test repository at least 20 times.
- Hash canonical JSON.
- Require exactly one hash per repository/configuration combination.
- Record runtime variance separately from result identity.

Cross-platform evaluation is desirable but may require a later environment. The
data model and tests should make platform differences visible even if only
Windows is available during initial implementation.

## 11. Evidence Rule Redesign

### 11.1 Problem

The current registry maps arbitrary strings to categories. The matcher infers
whether a key is a filename or path by checking for `/`. This is too ambiguous
for a trustworthy engine.

### 11.2 Proposed Rule Model

Conceptually:

```text
EvidenceRule
  ID
  Kind
  Pattern
  Category
  CasePolicy
  Scope
  Description
  WeightHint
```

Possible kinds:

- Exact filename
- Exact repository-relative path
- Exact directory name
- Path suffix by complete path segments
- Glob, only if necessary

Glob and regular-expression support should not be introduced without a concrete
need. Exact semantics are easier to reason about and test.

### 11.3 Case Policy

Case sensitivity should belong to the rule, not be inferred solely from the
operating system.

Possible policies:

- Exact
- Filesystem-aware
- ASCII case-insensitive

Markers such as `DESCRIPTION` may require exact matching even on a
case-insensitive filesystem to avoid semantic ambiguity. This decision must be
tested against realistic repositories.

### 11.4 Match Result

Every match should return:

```text
EvidenceMatch
  ruleID
  category
  matchedPath
  matchedName
  ruleKind
  rationale
```

The snapshot should preserve the stable rule ID rather than only the matched
filename.

### 11.5 Registry Correctness Tests

Tests should cover:

- Exact filename match
- Exact path match
- Exact directory match
- No partial-segment suffix match
- Windows-style and canonical separators
- Case-policy behavior
- Conflicting rules
- Duplicate rules
- Directory markers such as `.github/workflows`
- Conventional filenames including `.terraform.lock.hcl`
- Ambiguous filenames including `DESCRIPTION` and `.git/description`

### 11.6 Rule Conflicts

The engine must define what happens when multiple rules match:

- Prefer the most specific rule.
- Preserve all legitimate matches if they describe different facts.
- Reject invalid duplicate rule IDs during rule compilation.
- Use stable ordering for multiple matches.

The exact policy will be recorded in the decision log.

## 12. Noise, Exclusions, and Traversal Policy

### 12.1 Terminology

Phase 1 should use careful language:

- **Included:** Fully traversed and included in structural statistics.
- **Classified:** Traversed or observed and assigned a known operational role.
- **Pruned:** Directory entry observed, descendants not traversed.
- **Excluded from architecture:** Recorded but omitted from module inference.
- **Inaccessible:** Could not be read.

“Ignored” should be avoided unless its semantics are explicit.

### 12.2 Initial Policy Categories

Potential categories:

- Version-control metadata
- Installed dependency tree
- Build output
- Generated output
- Cache
- Coverage output
- IDE metadata
- Atlas output
- Vendored or third-party content
- Unknown

Not all categories should be pruned by default.

For example:

- `.git`: prune by default.
- `node_modules`: prune by default, record presence.
- `bazel-out`: prune by default.
- `dist`: context-dependent; classify carefully.
- `build`: context-dependent; must not be blindly pruned everywhere.
- `third_party`: do not prune by name alone.
- `vendor`: ecosystem-dependent; record and potentially traverse at a reduced
  level until semantics are decided.

### 12.3 Rules Versus Heuristics

Phase 1 traversal policy should favor high-confidence explicit rules.

Examples:

- Exact `.git` directory: deterministic prune rule.
- Exact `node_modules` directory: deterministic installed-dependency rule.
- Arbitrary directory named `build`: not automatically safe to prune without
  context.

Probabilistic noise classification belongs primarily to Phase 2.

### 12.4 Git-Aware Behavior

Potential sources:

- `.gitignore`
- `.git/info/exclude`
- Global Git ignore configuration
- `git ls-files`

Phase 1 should decide whether Git awareness is:

1. Disabled by default
2. Optional
3. Required for canonical behavior

Initial recommendation:

- Do not require invoking Git for the canonical default scan.
- Support a future explicit Git-aware policy.
- Record whether Git-aware behavior was enabled.

Requiring Git could make output depend on repository state, Git availability,
submodules, and global user configuration. That weakens reproducibility unless
carefully controlled.

### 12.5 Symlinks and Junctions

The policy must define:

- Whether directory symlinks are followed
- How Windows junctions are handled
- How cycles are prevented
- Whether linked content outside the root is permitted

Initial recommendation:

- Do not follow directory symlinks or junctions outside the scan root.
- Record encountered links as observations.
- Avoid duplicate traversal through alternate paths.

This recommendation must be confirmed against Go `WalkDir` behavior on the test
environment.

## 13. Scan Configuration

Canonical results must identify the configuration that produced them.

Potential fields:

```text
ScanConfiguration
  traversalPolicyVersion
  evidenceRuleSetVersion
  includeHidden
  followSymlinks
  enabledExclusionCategories
  userExclusions
  maximumDepth
```

Phase 1 should keep the user-facing CLI small. Configuration fields may exist in
the model before every field has a CLI flag.

Two scans with different policies must not be presented as directly identical.

## 14. Error Handling

### 14.1 Traversal Errors

Atlas should define whether an inaccessible path:

- Fails the complete scan
- Produces a partial scan with warnings
- Depends on strict mode

Initial recommendation:

- Default: complete the scan where possible and record inaccessible paths.
- Strict mode: fail on any traversal error.

This may require a custom traversal strategy because `WalkDir` currently stops
when the callback returns an error.

### 14.2 Output Errors

Report writing errors must no longer be discarded.

The CLI should:

- Return a nonzero exit code when requested output cannot be written.
- Clearly distinguish scan failure from report failure.
- Avoid leaving misleading partial artifacts where practical.

### 14.3 Error Ordering

Multiple warnings and inaccessible paths must be sorted canonically.

## 15. CLI Shape During Phase 1

The long-term command is:

```bash
atlas scan .
```

Phase 1 may begin formalizing this command. A possible minimum interface:

```bash
atlas scan <path>
atlas scan <path> --format json
atlas scan <path> --output <path>
atlas scan <path> --strict
```

CLI redesign is secondary to the snapshot and may be kept deliberately narrow.
Backward compatibility with the current positional invocation should be
considered, but the project is early enough that clarity may be preferable.

Any CLI change must be documented and tested.

## 16. Evaluation Harness

### 16.1 Purpose

The evaluation harness should answer:

- Is the result deterministic?
- How many paths were included, pruned, or inaccessible?
- Which evidence rules fired?
- Which regions were promoted by current heuristics?
- How long did the scan take?
- How much memory was used, if practical to measure?
- Did Atlas encounter obvious false positives?

### 16.2 Reproducibility

Each evaluation record should include:

- Repository identifier
- Repository commit hash, when available
- Atlas commit hash
- Snapshot schema version
- Rule-set version
- Scan configuration
- Canonical snapshot hash
- Number of repeated runs
- Runtime summary

The repository commit hash is evaluation metadata. It should not be required for
normal scanning.

### 16.3 Stored Results

Recommended layout:

```text
codexPlans/
  phase-1-deterministic-foundation.md
  decisions.md
  results/
    phase-1/
      summary.md
      selenium.metrics.json
      tensorflow.metrics.json
      vscode.metrics.json
```

Full repository snapshots may be large and machine-specific. Before committing
them, inspect their size and content. Prefer committed metrics, hashes, and
curated findings. Large raw artifacts should go to an ignored evaluation-output
directory.

### 16.4 Baseline Before Modification

Before Phase 1 implementation changes behavior, record the current output for all
three repositories:

- Runtime
- Total files
- Evidence totals
- Top directories
- Candidate modules
- Retained modules
- Obvious false positives
- Repeated-run output differences

This baseline will show what Phase 1 corrected.

## 17. Testing Strategy

### 17.1 Unit Tests

Add focused tests for:

- Rule compilation
- Rule matching
- Rule specificity and conflicts
- Path normalization
- Canonical sorting
- Canonical serialization
- Exclusion policy
- Traversal pruning
- Snapshot immutability during reporting
- Configuration identity
- Error collection

### 17.2 Fixture Tests

Create small repository fixtures representing:

- Single-module Go repository
- JavaScript repository with `node_modules`
- Bazel repository with `bazel-out`
- Polyglot repository
- Nested package manifests
- Ambiguous marker casing
- Directory evidence such as `.github/workflows`
- Inaccessible path, where test environment permits
- Symlink or junction behavior
- Equal-score module candidates

Fixtures should be small enough to understand manually.

### 17.3 Golden Tests

Canonical JSON and text output are good candidates for golden tests.

Golden tests must:

- Use intentionally small fixtures.
- Avoid absolute paths and timestamps.
- Be easy to review.
- Fail clearly when the schema changes.
- Require deliberate updates.

### 17.4 Integration Tests

- Repeated scans produce identical bytes.
- Different output renderers consume the same snapshot.
- Output failures return appropriate errors.
- A pruned directory is recorded but not traversed.
- The scan does not write inside the analyzed repository unless explicitly
  requested.

### 17.5 Performance Tests

Phase 1 should establish baselines rather than hard optimization targets.

Measure:

- Wall-clock duration
- Files observed per second
- Peak memory if practical
- Snapshot size
- Evidence matching cost

Any redesign that severely regresses scans of TensorFlow should be investigated
before Phase 1 completion.

## 18. Work Breakdown

### Workstream 1: Baseline and Characterization

- Run current Atlas against all three repositories.
- Capture concise baseline metrics.
- Repeat scans to expose nondeterministic output.
- Catalogue false-positive evidence and noise promotion.
- Record repository commit identities.

Deliverable:

- `codexPlans/results/phase-1/baseline.md`

### Workstream 2: Canonical Model

- Define snapshot schema.
- Separate canonical data from runtime metadata.
- Define canonical paths.
- Define ordering rules.
- Add schema and rule-set versions.
- Implement canonical JSON serialization.

Deliverable:

- Versioned snapshot model and golden tests.

### Workstream 3: Evidence Engine

- Replace ambiguous registry map with typed evidence rules.
- Define exact matching semantics.
- Add stable rule IDs.
- Add case policy.
- Add conflict handling.
- Port existing evidence definitions.
- Correct invalid or misspelled definitions.
- Add comprehensive matcher tests.

Deliverable:

- Explainable evidence matches with regression coverage.

### Workstream 4: Traversal Policy

- Define high-confidence prune rules.
- Record pruned regions.
- Handle `.git`, `node_modules`, and known generated build trees.
- Define symlink and junction behavior.
- Define inaccessible-path behavior.
- Ensure pruning affects statistics consistently.

Deliverable:

- Policy-controlled traversal with explicit exclusion records.

### Workstream 5: Deterministic Aggregation

- Sort all canonical slices.
- Add complete tie-breakers.
- Remove map-order dependence from reports.
- Prevent renderers from mutating snapshot data.
- Normalize paths.
- Ensure current module and hierarchy inference consume stable inputs.

Deliverable:

- Byte-stable canonical and text outputs.

### Workstream 6: CLI and Persistence

- Formalize scan command behavior.
- Separate canonical output from runtime report envelope.
- Make output paths explicit.
- Handle write failures.
- Avoid automatic writes that contaminate subsequent scans.
- Preserve a sensible compatibility path for current usage.

Deliverable:

- Predictable CLI behavior and tested exit codes.

### Workstream 7: Evaluation

- Run at least 20 scans per test repository.
- Compare snapshot hashes.
- Record performance.
- Review evidence and exclusions.
- Document current heuristic output after noise correction.
- Identify remaining accuracy work for Phase 2.

Deliverable:

- `codexPlans/results/phase-1/summary.md`
- Concise per-repository metric files.

## 19. Proposed Implementation Sequence

The recommended order is:

1. Capture current baseline.
2. Define canonical snapshot types and ordering.
3. Add canonical serializer and small golden fixture.
4. Introduce typed evidence rules.
5. Port current registry and add matcher tests.
6. Introduce traversal policy and exclusion records.
7. Refactor collection to emit canonical observations.
8. Make existing module inference consume the new observations.
9. Make text reporting consume the canonical snapshot.
10. Formalize CLI output behavior.
11. Add repeated-run determinism tests.
12. Evaluate Selenium, TensorFlow, and VS Code.
13. Fix Phase 1 regressions.
14. Complete the phase report.

Each step should leave tests passing. Large behavior changes should not be
combined into one patch.

## 20. Decision Gates Requiring Review

The following decisions should be reviewed before or during implementation:

1. Should canonical snapshots include the normalized root name?
2. Should default scans prune `node_modules` and Bazel output completely?
3. How should Atlas report file counts for pruned directories?
4. Should inaccessible paths fail by default or produce partial results?
5. Should `atlas scan` replace the current positional CLI immediately?
6. Should Git ignore rules influence default behavior?
7. Should current module inference remain in the Phase 1 snapshot or be exposed
   as a legacy inference section?
8. What versioning convention should rule sets use?
9. Should canonical JSON be intended as a public API in Phase 1 or remain
   explicitly experimental?

Decisions will be recorded in `codexPlans/decisions.md`.

## 21. Risks

### Risk: Overengineering the Foundation

Mitigation:

- Use concrete needs from the three repositories.
- Avoid plugin frameworks and generic rule languages.
- Keep typed rules simple.

### Risk: Treating Common Directory Names as Universal Noise

Mitigation:

- Prune only high-confidence operational directories in Phase 1.
- Record every prune decision.
- Defer probabilistic classification to Phase 2.

### Risk: Canonical Output Becomes Machine-Specific

Mitigation:

- Use repository-relative paths.
- Separate runtime metadata.
- Test equivalent fixtures under different roots.

### Risk: Correctness Changes Break Existing Heuristics

Mitigation:

- Capture baseline first.
- Treat changed conclusions caused by removed noise as expected.
- Separate observation correctness from inference accuracy.

### Risk: TensorFlow Performance Regresses

Mitigation:

- Measure before and after.
- Compile rules once.
- Avoid repeated full-tree scans.
- Avoid storing unnecessary per-file data.

### Risk: Test Corpus Bias

Mitigation:

- State clearly that three repositories are an initial corpus.
- Do not tune rules exclusively to their directory names.
- Add synthetic fixtures for general behavior.

## 22. Trust Model

Phase 1 establishes the first three reasons to trust Atlas.

### Determinism

The result is reproducible and stable.

### Traceability

Every evidence or exclusion decision names the rule that produced it.

### Honest Scope

Atlas distinguishes measured observations from provisional heuristics.

Phase 2 will add measured classification accuracy. Phase 3 will add evidence-
backed recommendations.

## 23. Exit Criteria

Phase 1 is complete only when all required criteria pass.

### Determinism

- Twenty unchanged scans of each test repository produce one canonical hash.
- Text reports are stable when runtime metadata is excluded.
- Equivalent fixtures under different absolute roots produce equivalent
  canonical snapshots.
- Every sort has a documented deterministic tie-breaker.

### Evidence Correctness

- Typed rules replace ambiguous registry inference.
- File and directory matching semantics are separate.
- `.git/description` does not match the R `DESCRIPTION` rule.
- Directory rules match only intended path segments.
- Every evidence match contains a stable rule ID.
- Evidence tests cover ambiguity and conflict behavior.

### Traversal and Noise

- `.git` is not traversed by default.
- `node_modules` is not promoted as a product/module region.
- Known Bazel output trees are not promoted as product/module regions.
- Every pruned path is represented in the snapshot.
- User-visible output explains default exclusions.

### Snapshot and Reporting

- Canonical snapshot schema is versioned.
- Canonical output excludes timestamps and machine-specific absolute paths.
- Text and JSON consume the same snapshot.
- Renderers do not mutate the snapshot.
- Output write failures are reported.

### Evaluation

- Baseline and final metrics exist for Selenium, TensorFlow, and VS Code.
- Runtime regressions are understood and documented.
- Remaining major false positives are documented for Phase 2.
- Current heuristic behavior is characterized rather than presented as proven
  architectural truth.

### Quality

- `go test ./...` passes.
- `go vet ./...` passes.
- New critical paths have focused tests.
- No production source file becomes an oversized mixed-responsibility module.

## 24. Definition of Done Document Update

Before Phase 1 is marked complete, this document must be updated with:

- Final architecture
- Decisions made
- Work completed
- Deviations from plan
- Test results
- Repository evaluation results
- Performance measurements
- Known limitations
- Deferred work for Phase 2
- Final exit-criteria checklist

The status at the top must then change from `Planned` to `Completed`.

## 25. Phase 2 Handoff Requirements

Phase 2 planning should not begin until it can consume:

- Stable canonical snapshots
- Typed evidence matches
- Explicit exclusions
- Baseline module inference results
- Evaluation metrics from all three repositories
- A list of known classification errors

Phase 2 will focus on:

- Structural roles
- Signal versus noise beyond explicit rules
- Multidimensional scoring
- Confidence
- Supporting and counter-evidence
- Major-region precision and recall

## 26. Approval Checkpoint

No Phase 1 implementation should begin until this plan has been reviewed.

Review should focus on:

- Whether the phase is too broad or too narrow
- Whether pruned directories should retain estimated counts
- Whether Git-aware behavior belongs in Phase 1
- Whether canonical JSON should be public or experimental
- Whether the exit criteria reflect the level of trust Atlas needs

After approval, implementation begins with baseline scans of the three test
repositories.
