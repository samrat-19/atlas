# Heuristic Calibration Notes

Atlas currently uses numeric heuristic defaults such as:

- large folder threshold: 200 files
- evidence-dense folder threshold: 20 files
- evidence density threshold: 5%
- module evidence weight: 100
- compression evidence weight: 200
- child retention ratio: 60%
- novelty threshold: 20%
- high-overlap threshold: 90%

These numbers are useful starting assumptions, not proven truths.

## Why This Matters

The numbers are understandable by intuition, but Atlas should not pretend they
are universally correct. They encode rough beliefs such as:

- a folder with many files is probably worth noticing
- a build or package marker matters more than an ordinary file
- a child folder should survive compression if it is strong enough or different
  enough from its parent
- a child folder that looks almost identical to its parent may be redundant

That is acceptable for a prototype only if Atlas is transparent that these are
defaults that need calibration.

## How We Should Justify Numbers Later

Atlas should calibrate heuristic defaults against real repositories.

Initial calibration repositories:

- Selenium
- TensorFlow
- VS Code

For each repository, we should define expected outcomes:

- important regions Atlas should promote
- external or noisy regions Atlas should not over-promote
- generated or fixture regions Atlas should recognize as low priority
- top-ranked results that an engineer would actually inspect first

Then we can ask:

- Did the current numbers rank expected regions?
- What false positives appeared?
- What important regions were missed?
- Did a threshold change improve one repository while hurting another?

This turns the question from "why 200?" into:

> We use this value because it performed best against known repositories without
> flooding the result with noise.

## Code Manageability Direction

**Status: done (Phase 2 D2).** `HeuristicProfile` now lives in
`internal/model/heuristics.go` (moved here from `internal/collector` during
the Phase 2 package-boundary refactor — see `docs/phase-2-plan.md`), grouped
into `EvidenceConfidenceConfig`,
`CandidateSelectionConfig`, `ScoringConfig`, and `CompressionConfig`.
`DefaultHeuristics` holds the same numbers Phase 1 used as bare constants —
nothing about scoring changed, only where the numbers live. The profile is
passed as an explicit parameter through `buildModuleSummary`, `scoreModules`,
`compressModules`, and `MatchEvidence`'s path-context confidence discount,
not read from a package-level global inside those functions — so a future
alternate profile only requires constructing a `HeuristicProfile` value and
passing it to `Collect`, not editing scoring logic.

One deliberate deviation from the sketch above: there is no `ReportConfig`.
`heuristics.go` governs the classification model (which folders qualify, how
they're scored and compressed); how many results the CLI prints
(`report_limits.go`) is a display concern, not a classification heuristic,
and folding it into the same profile would conflate two different things
that happen to both be "numbers." It can be added if a real need for
profile-driven display limits shows up.

This keeps numbers centralized and makes it easier to introduce future modes:

- default
- large monorepo
- dependency investigation
- onboarding
- strict noise reduction
- AI context planning

## Longer-Term Direction

Atlas should eventually move away from one magic score and toward named,
explainable dimensions:

- boundary confidence
- size prominence
- evidence strength
- noise probability
- first-party probability
- novelty versus parent
- investigation priority

This would let Atlas explain results like:

```text
Boundary confidence: high
Size prominence: medium
Noise probability: low
Novelty: high
Investigation priority: high
```

That is easier to trust than a single unexplained score.

## Recommendation

Short term:

- keep current constants
- document them as defaults, not truths
- use battery tests to catch output drift

Medium term — **done (Phase 2 D2)**:

- introduce `HeuristicProfile`
- move defaults into `DefaultHeuristics`
- pass the profile through scoring and compression
- add tests for profile behavior

Long term:

- split scoring into multiple explainable dimensions
- calibrate defaults against real repositories
- record why defaults were chosen
- support task-specific profiles
