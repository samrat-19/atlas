# Atlas Heuristics

Atlas is trying to answer a practical question:

> In this repository, which folders are worth looking at first?

Today Atlas answers that with simple structural rules. These rules are not final
truth. They are the current baseline that lets Atlas turn a large directory tree
into a shorter orientation report.

Atlas does not read source code. It looks at structure:

- file counts
- file extensions
- build files
- package files
- CI files
- container files
- where those files appear in the tree

## The Current Pipeline

Atlas uses three steps:

1. Pick candidate folders.
2. Score those candidates.
3. Compress the candidate list.

## 1. Candidate Folders

Atlas first asks:

> Should this folder be considered at all?

A folder becomes a candidate if any of these are true.

## Rule 1: The Folder Has Evidence

If a folder contains an evidence file, Atlas considers it.

Examples of evidence:

- `package.json`
- `go.mod`
- `BUILD`
- `BUILD.bazel`
- `WORKSPACE`
- `Dockerfile`
- `.github/workflows`

Example:

```text
javascript/package.json
```

The `javascript` folder is interesting because it contains package-manager
evidence.

## Rule 2: The Folder Is Large

If a folder has at least 200 files, Atlas considers it even if it has no evidence
file.

Why? Some important source trees do not have their own build file at the folder
root. Size alone is not proof of importance, but a large folder is usually worth
noticing.

Example:

```text
src/       500 files
```

Atlas should not ignore `src/` just because the build file lives somewhere else.

## Rule 3: The Folder Has High Evidence Density

If a folder has at least 20 files, and at least 5% of them are evidence files,
Atlas considers it.

Example:

```text
folder has 20 files
1 file is BUILD.bazel
1 / 20 = 5%
```

That is enough to make the folder interesting.

Important note: because "has any evidence" already qualifies a folder, this rule
overlaps with Rule 1 today. We keep it visible because later versions may treat
evidence density differently from evidence presence.

## 2. Candidate Score

After Atlas has candidates, it gives each one a rough score.

The first score is:

```text
score = evidence_count * 100 + file_count
```

Meaning:

- evidence matters a lot
- size also matters
- one evidence file counts like 100 ordinary files

Example:

```text
folder A: 1 evidence file, 20 files
score = 1 * 100 + 20 = 120

folder B: 0 evidence files, 90 files
score = 0 * 100 + 90 = 90
```

Folder A ranks higher because the evidence file suggests it may be a project or
build boundary.

## 3. Compression

Large repositories can produce hundreds or thousands of candidates. Atlas then
tries to shorten the list.

Compression asks:

> Should we keep this child folder, or is its parent enough?

Atlas looks at three things.

## Strength

A child folder can survive if its score is at least 60% of its parent's score.

Example:

```text
parent score: 1000
child score:  650
```

The child is strong enough to keep.

## Novelty

A child folder can survive if it looks different from its parent.

Atlas compares:

- common file extensions
- evidence categories

Example:

```text
parent: mostly .java
child:  mostly .ts
```

The child probably represents a different subsystem, so Atlas keeps it.

Today "different enough" means more than 20% different.

## Redundancy

If a child looks almost the same as its parent, Atlas penalizes it.

Example:

```text
parent: mostly .java, build-system evidence
child:  mostly .java, build-system evidence
```

That child may be a smaller part of the same area rather than a separate major
module.

Today "almost the same" means 90% overlap or higher for both:

- extension mix
- evidence category mix

When that happens, Atlas subtracts 500 points from the child score.

## Compression Score

Compression uses a second score:

```text
score = evidence_count * 200 + file_count + coverage_percent * 10
```

This gives credit for:

- evidence files
- local size
- how much of the repository lives under that folder

Evidence gets a higher weight during compression because Atlas is choosing which
candidate folders deserve to survive into the final shorter list.

## Current Limitations

These heuristics are intentionally rough.

Known problems:

- A single `BUILD` file can make a tiny folder look important.
- Bazel repositories can produce too many candidate modules.
- Generated files can rank highly because of file count.
- `third_party` can look important before Atlas understands first-party versus
  external code.
- Test fixtures can look like real modules when they contain package files.
- The current score mixes several ideas: size, evidence, coverage, and novelty.

This is why Phase 2 should split the model into clearer dimensions. **Status
as of Phase 2 D3:** `ModuleCandidate` now carries `BoundaryConfidence`,
`EvidenceStrength`, `StructuralProminence`, `NoveltyVsParent`, and
`NoiseProbability` as named 0–1 fields, computed from D1's per-match
confidence and shown directly in the "Major Modules" report section
alongside the legacy score. See the doc comment on `ModuleCandidate` in
`internal/collector/types.go` for exactly what each one means and how it's
computed.

**Status as of Phase 2 D4a:** `ModuleCandidate` now also carries `Role` — a
label, not a number, computed in `role.go` from a fixed, ordered list of
directory-name pattern checks (`patterns.go`) plus the `EvidenceStrength`
fallback above. `third_party` directories are now explicitly labeled
`vendored` rather than just scored lower; the same goes for `test-fixture`
(reusing D1's noise-adjacent path signal) and `generated` (added after the
unrecognized-extension diagnostic — see `unrecognized.go` — found a real,
unexplained 504-file cluster at a VS Code path literally containing
"generated"). Candidates with no evidence and no path match are labeled
`ambiguous` rather than guessed — confirmed against TensorFlow's golden
API-definition and security-advisory folders, which is exactly what they
get labeled.

This substantially addresses "first-party probability" from the original
wishlist below, as a label (`RoleFirstParty`) rather than a separate
probability number. `Role` is currently informational only — it does not
affect `Score`, retention, or compression. Whether it should is a separate,
deliberately deferred decision (Phase 2 D4b — see `docs/phase-2-plan.md`).

Still open — not yet split out:

- evidence diversity
- investigation priority

These need more than a per-directory confidence number or a path-pattern
label to answer honestly, and don't yet have a concrete design.

## Why Keep These Heuristics?

They are simple, fast, and explainable.

They give Atlas a useful first pass without needing source-code parsing,
dependency resolution, or AI.

The goal is not for these numbers to be perfect. The goal is to make the current
rules visible, testable, and easy to improve.
