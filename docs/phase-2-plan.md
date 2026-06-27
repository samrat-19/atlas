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

### D4: Structural-role classification

Using D1–D3, classify candidate regions into roles: first-party root,
vendored/third-party, generated, test-fixture, build-output, ambiguous. This
is where `build`, `dist`, `vendor`, `third_party` finally get resolved
per-directory instead of the current all-or-nothing prune list — and where
"ambiguous" becomes a legitimate, explicit output rather than a forced guess.

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
