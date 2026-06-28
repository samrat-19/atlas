# Phase 2 Plan: Confidence, Role, and Explainable Scoring

## Purpose

Phase 1 made Atlas's observations deterministic and reproducible. Phase 2 makes
its judgments explainable.

Today a single integer score collapses size, evidence weight, and
parent-similarity into one number. There is no way to ask "how confident is
Atlas that this is a real boundary" separately from "how big is it."
`docs/heuristics.md` and `docs/heuristic-calibration.md` already name the
target shape — named, explainable dimensions instead of one score. Phase 2
builds toward that.

## Goal

Give Atlas the data and structure to answer, per candidate region:

- How strong is the evidence here, and how much of it is first-party versus
  incidental?
- Does this look like a real project/build boundary, a vendored/generated/
  test-fixture area, or something Atlas genuinely isn't sure about?
- When it isn't sure, say so explicitly, instead of forcing a guess.

This directly targets the limitations already logged in `docs/heuristics.md`:
tiny folders inflated by a single build file, `third_party` scored the same as
first-party code, test fixtures with package files ranked like real modules.

## Deliverables (in dependency order)

### D1: Typed evidence rules

Replace the flat `EvidenceRegistry map[string]string` with typed rules that
carry, per entry: a stable ID, a category, and a confidence weight (e.g. "this
filename alone is strong signal" vs. "this filename is weaker signal near
known noise-adjacent paths like `test/`, `fixtures/`, `examples/`").
`MatchEvidence` returns the richer match; `EvidenceItem` carries it through
collection.

This is the foundation everything below reads from — role classification and
multidimensional scoring both need confidence data to exist before they can
use it.

### D2: HeuristicProfile — done

Centralize the constants currently hardcoded in `heuristics.go` — plus the new
confidence weights from D1 — into a single `HeuristicProfile` struct, per the
shape already sketched in `docs/heuristic-calibration.md`. The default profile
must reproduce Phase 1 behavior exactly; this step relocates where the numbers
live, it does not change scoring yet.

Landed as a `HeuristicProfile` struct (`EvidenceConfidenceConfig`,
`CandidateSelectionConfig`, `ScoringConfig`, `CompressionConfig`) with
`DefaultHeuristics` holding the unchanged Phase 1 values. The profile is
threaded as an explicit parameter through `MatchEvidence`,
`buildModuleSummary`, `scoreModules`, and `compressModules` — not read from
package constants inside those functions — confirmed by
`TestIsModuleCandidateRespectsCustomProfile` and
`TestIsStrongComparedToParentRespectsCustomProfile`, which construct an
alternate profile and assert behavior actually changes. No `ReportConfig`
was added (see `docs/heuristic-calibration.md` for why). Battery output for
selenium and tensorflow is byte-identical to before D2.

### D3: Multidimensional scoring — done

Split the single `Score int` on `ModuleCandidate` / `RegionNode` into named,
explainable dimensions: boundary confidence, evidence strength, size
prominence, novelty vs. parent, noise probability. Computed from D1's
confidence data and D2's profile. A derived single score can stay for
backward compatibility if useful, but the dimensions become the primary
output.

Landed on `ModuleCandidate` as `BoundaryConfidence`, `EvidenceStrength`,
`StructuralProminence` (renamed from "size prominence" to match
`docs/heuristics.md`'s existing wording), `NoveltyVsParent`, and
`NoiseProbability` — all 0–1, all computed deterministically, none involving
AI. `EvidenceStrength`/`NoiseProbability` only need a candidate's own stats
(`modules.go`); `StructuralProminence`/`NoveltyVsParent`/`BoundaryConfidence`
need subtree and parent context, so they're computed in `scoreModules`
(`module_scoring.go`) where that context already exists. `Score` is
unchanged, kept for backward-compatible sorting. All five are now printed in
the "Major Modules" report section — confirmed sane against real noise
patterns in tensorflow (`third_party` directories with no evidence get a
neutral 0.5 `NoiseProbability`, not a falsely confident one; evidence under
`test/` paths visibly shows the D1 confidence discount).

Deliberately deferred: propagating these dimensions into `RegionNode` /
the hierarchy view. Aggregating five non-additive 0–1 scores across a
subtree (max? weighted average?) is a distinct design problem from computing
them per-candidate, and doesn't yet have a justified answer — `RegionNode`
keeps only `Score` for now.

Also deferred, per `docs/heuristics.md`'s own wishlist: first-party
probability, evidence diversity, investigation priority. First-party
probability in particular overlaps with D4's structural-role classification
rather than being a separate dimension — better resolved there.

### D4 prerequisite: unrecognized-extension diagnostic — done

Before writing D4's role-classification rules from guessed conventions,
built a small, additive diagnostic (`UnrecognizedSummary`,
`buildUnrecognizedSummary()` in `unrecognized.go`) that groups module
candidates with zero evidence (qualified purely by size) by shared dominant
extension, and prints them in a new "Unrecognized Extension Clusters" report
section. Changes no scoring, classification, or retention behavior — purely
observational, same low-risk shape as D1/D3.

Run against all three battery repos, this found only 5 unrecognized
directories total: 0 in Selenium, 3 in TensorFlow (`.pbtxt` golden
API-definition files, a `.md`-heavy security advisory folder), 1 in VS Code
(`.ts` files at a path literally containing `generated`). None point to a
missing `EvidenceRegistry` rule (no missing build file or package manager) —
the gaps are about **role**, not **evidence**, which directly confirms D4's
plan to add a `generated` path-pattern rule is grounded in an observed real
case, not a guess. It also confirms the existing registry already explains
the large majority of large directories across three structurally different
repos — Atlas's blind spots are narrow, not pervasive.

### D4a: Structural-role labeling — done

Using D1–D3, classify candidate regions into roles: first-party, vendored,
generated, test-fixture, build-output, ambiguous. This is where `vendor`,
`third_party`, and generated-code paths finally get resolved per-directory
instead of the current all-or-nothing prune list — and where "ambiguous"
becomes a legitimate, explicit output rather than a forced guess.

Landed as `Role` on `ModuleCandidate` (`role.go`), computed by
`classifyRole()`: a fixed, ordered list of checks — vendored path pattern,
then generated, then build-output, then test-fixture (reusing D1's
noise-adjacent segments), then an evidence-strength threshold for
first-party, then ambiguous as the explicit default. The order is the
determinism guarantee: a path matching two pattern sets at once always
resolves to whichever check is listed first, never to map-iteration order —
proven by `TestClassifyRolePrecedenceIsFixedNotMapOrder`. The pattern data
itself moved into a new `patterns.go` ("knowledge layer"), consolidating
what used to be one map scattered in `registry.go` into one documented home
alongside the new vendored/generated/build-output sets.

`buildOutputPathSegments` deliberately excludes `build`, `bin`, and `out` —
ambiguous in some ecosystems (the same reason `prune.go` never hard-pruned
`build`) — keeping only `dist` and `target`. Being wrong about a genuine
source directory is worse than missing a build-output label.

Verified against real output, not just unit tests: every `vendored` label
across Selenium and TensorFlow is genuinely under a `third_party`/`vendor`
path; TensorFlow's 3 `ambiguous` labels are exactly the 3 directories the
D4 prerequisite diagnostic found (no path pattern, no evidence — an honest
answer, not a guess); VS Code's one `generated` label is exactly the real
case that motivated adding the pattern in the first place.

`Role` is a label only — it does not affect `Score`, retention, or
compression, and is not propagated to `RegionNode` (same deferral reasoning
as D3's dimensions: no settled answer yet for aggregating a label across a
subtree). Whether a role like `vendored` or `build-output` should ever
change retention is **D4b**, a separate, deliberately unmade decision —
see below.

### D4b: Whether roles affect retention — not started

Open design question, intentionally not decided in D4a: should a
`vendored`/`build-output`/`generated` label actually exclude a candidate
from the major-modules list, or stay purely informational? This touches the
same compression/retention logic already patched once for the negative-score
bug, so it needs its own deliberate review and a calibration pass — not a
decision made as a side effect of adding the label.

### D5: Calibration pass

Re-run against Selenium, TensorFlow, and VS Code. Per
`docs/heuristic-calibration.md`'s methodology: define expected
promotions/demotions per repo before looking at output, then check for false
positives, missed regions, and whether a change that helps one repo hurts
another. Update battery fixtures only after expected outcomes are met — not
just because output changed.

## Constraints

- Stay fully deterministic. Confidence and role are computed from structure,
  not inferred by AI. Where structure genuinely underdetermines the answer,
  the output is "ambiguous," not a coin-flip guess.
- No new packages, no CLI redesign, no `ScanConfiguration` exposed to users,
  no AI integration. Those are Phase 3 or later — Phase 2 is about the shape
  of the judgment itself, not how it's configured or who consumes it.

## Out of Scope

- User-facing configuration of evidence rules or profiles
- Git-aware traversal, symlink/junction policy
- AI-assisted classification of any kind
- CLI surface changes

## Exit Criteria

- `EvidenceRegistry` is typed rules with confidence, not a flat string map.
- `HeuristicProfile` exists; the default profile reproduces Phase 1 battery
  output exactly before any dimension changes are introduced.
- `ModuleCandidate` / `RegionNode` expose named dimensions, not just one
  score.
- At least `build`, `dist`, `vendor`, `third_party` are classified
  per-directory rather than handled by the binary prune list.
- Battery tests pass against updated, deliberately-reviewed expected fixtures
  for all three repositories.
- `go test ./...` and `go vet ./...` pass throughout.
