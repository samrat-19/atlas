package collector

// schemaVersion is embedded in every Result so consumers can detect breaking
// changes to the JSON shape. The rule:
//   - Adding an optional field → no bump needed.
//   - Removing a field or changing its meaning → increment the number.
//
// Phase 1 establishes "1" as the first formally versioned schema.
const schemaVersion = "1"

// Atlas currently uses a simple three-step heuristic:
//
// 1. Candidate selection:
//    Decide which folders are worth considering as possible modules.
//
// 2. Candidate scoring:
//    Rank those folders using a rough score based on evidence and size.
//
// 3. Compression:
//    Keep useful candidates and drop nested candidates that look redundant with
//    their parent.
//
// These rules are intentionally simple. They are not a final definition of
// repository importance. They are the current Phase 1 baseline that helps Atlas
// turn a large directory tree into a shorter orientation report. See
// docs/heuristics.md for the full walkthrough with examples.
//
// Phase 2 D1 added a fourth concern — evidence confidence — that adjusts how
// strong a match is before any of the three steps above run.
//
// Phase 2 D2 (this file) moves every tunable number out of bare package
// constants and into a single HeuristicProfile value (DefaultHeuristics),
// grouped by which step reads them. The grouping and the numbers are
// unchanged from Phase 1 — this only changes where they live, so a future
// profile can be constructed and passed through scoring and compression
// without touching the functions that use it. See
// docs/heuristic-calibration.md for the calibration methodology and the
// longer-term plan to split scoring into named, explainable dimensions
// instead of one number.

// EvidenceConfidenceConfig answers: "How strong is an evidence match on its
// own, before any scoring happens?" Read by MatchEvidence (registry.go).
type EvidenceConfidenceConfig struct {
	// NoiseAdjacentConfidenceMultiplier is applied to a rule's intrinsic
	// confidence when the match is found under a noise-adjacent directory
	// (test/fixtures/examples/mocks — see noiseAdjacentPathSegments in
	// registry.go). A package.json under testdata/ is far less likely to
	// mark a real module boundary than one at a project root, but Atlas
	// cannot rule it out entirely, so the signal is weakened rather than
	// discarded.
	NoiseAdjacentConfidenceMultiplier float64
}

// CandidateSelectionConfig answers: "Should Atlas consider this folder at
// all?" Read by isModuleCandidate and its helpers (modules.go).
type CandidateSelectionConfig struct {
	// LargeDirectoryFileThreshold: a folder with this many files is worth
	// considering even if it has no obvious build file. Example: a large
	// src/ tree may matter even without its own package.json or BUILD file.
	LargeDirectoryFileThreshold int

	// EvidenceDenseFileThreshold: the "evidence density" rule only applies
	// to folders with at least this many files. This avoids overreacting to
	// tiny folders where 1 marker out of 1 file would look like "100%
	// evidence".
	EvidenceDenseFileThreshold int

	// EvidenceDensityThreshold: if at least this share of a folder's files
	// are evidence files, the folder is probably structurally interesting.
	// 0.05 means 5%, so 1 evidence file in 20 files is enough.
	EvidenceDensityThreshold float64
}

// ScoringConfig answers: "How strong does this candidate look, on first
// pass?" Read by newModuleCandidate and moduleCandidateScore (modules.go).
type ScoringConfig struct {
	// DominantExtensionLimit: for each candidate folder, remember only this
	// many common extensions. This gives a small fingerprint like
	// [".ts", ".json", ".md"] instead of the full file list.
	DominantExtensionLimit int

	// ModuleEvidenceWeight: first-pass ranking. One evidence file counts
	// like this many normal files. This makes a folder with package/build
	// files rank above a folder that is only slightly larger.
	ModuleEvidenceWeight int
}

// CompressionConfig answers: "When Atlas shortens the candidate list, what
// should survive?" Read by scoreModules and compressModules
// (module_scoring.go, module_compression.go).
type CompressionConfig struct {
	// CoveragePercentScale converts "files in this subtree / files in repo"
	// into a percentage. Example: 250 files out of 1000 becomes 25.
	CoveragePercentScale int

	// CompressionEvidenceWeight: evidence matters even more here than
	// ScoringConfig.ModuleEvidenceWeight, because Atlas is deciding which
	// candidate folders survive into the shorter "major modules" list.
	CompressionEvidenceWeight int

	// CoverageScoreWeight: during compression, each 1% of repo coverage adds
	// this many points. This gives large subtrees some credit without
	// letting size be the only signal.
	CoverageScoreWeight int

	// HighOverlapThreshold: if a child folder is this similar to its parent,
	// Atlas treats it as probably redundant. Example: parent and child are
	// both mostly .java and both only have build-system evidence.
	HighOverlapThreshold float64

	// RedundantChildPenalty: score removed from a child folder when it looks
	// almost the same as its parent. This nudges Atlas toward keeping the
	// parent instead of every nested folder with the same shape.
	RedundantChildPenalty int

	// ChildScoreRetentionRatio: keep a child folder if its score is at least
	// this fraction of its parent's score. 0.6 means a child that is 60% as
	// strong as the parent can survive.
	ChildScoreRetentionRatio float64

	// NoveltyRetentionDelta: keep a child folder if it is different enough
	// from its parent. 0.2 means "more than 20% different" by extension mix
	// or evidence category.
	NoveltyRetentionDelta float64
}

// DiagnosticsConfig answers: "How much detail should diagnostic summaries
// keep?" These numbers don't affect scoring, classification, or retention —
// they only bound how much data a diagnostic (like UnrecognizedSummary)
// retains for display. Read by buildUnrecognizedSummary (unrecognized.go).
type DiagnosticsConfig struct {
	// ExampleDirectoryLimit caps how many example paths
	// UnrecognizedExtensionCluster keeps per cluster. A pattern that recurs
	// across thousands of directories only needs a few examples for a human
	// to go look at — keeping all of them would bloat the JSON output for
	// no benefit.
	ExampleDirectoryLimit int
}

// RoleClassificationConfig answers: "Once path patterns don't confidently
// resolve a candidate's structural role, how strong does its own evidence
// need to be to call it first-party rather than ambiguous?" Read by
// classifyRole (role.go).
type RoleClassificationConfig struct {
	// FirstPartyEvidenceStrengthThreshold: a candidate with evidence at or
	// above this average confidence, and no vendored/generated/build-output/
	// test-fixture path match, is classified first-party. Below it, Atlas
	// has evidence but isn't confident enough in it to commit to a role, so
	// the candidate is ambiguous instead. With today's evidence registry
	// (every default rule at full intrinsic confidence — see registry.go)
	// this threshold mostly separates "has any evidence at all" from "has
	// none," since the only thing that currently lowers EvidenceStrength
	// below 1.0 is the same noise-adjacent discount that already routes a
	// candidate to test-fixture before this check is ever reached. It is
	// written as a real threshold, not a bare non-zero check, so it stays
	// meaningful once a future evidence rule has its own lower intrinsic
	// confidence for a reason other than path context.
	FirstPartyEvidenceStrengthThreshold float64
}

// HeuristicProfile bundles every tunable number Atlas's classification
// pipeline reads, grouped by the pipeline stage that consumes them. A
// profile is plain data: constructing a different one and passing it through
// Collect would change Atlas's behavior without changing any function body.
//
// Only DefaultHeuristics exists today. Nothing yet constructs an alternate
// profile or exposes one to users — that is out of scope for Phase 2 (see
// the constraints in docs/phase-2-plan.md). This type exists now so that the
// later work is additive (new profiles, a way to choose one) instead of a
// rewrite of how scoring reads its numbers.
type HeuristicProfile struct {
	EvidenceConfidence EvidenceConfidenceConfig
	CandidateSelection CandidateSelectionConfig
	Scoring            ScoringConfig
	Compression        CompressionConfig
	Diagnostics        DiagnosticsConfig
	RoleClassification RoleClassificationConfig
}

// DefaultHeuristics is the profile Atlas has always used. Its values are
// starting assumptions calibrated by hand against Selenium, TensorFlow, and
// VS Code (see docs/heuristic-calibration.md) — not proven-correct
// constants. Every number here is identical to the bare constants Phase 1
// used; D2 only changed where they live, not what they equal, so existing
// battery fixtures keep passing unchanged.
var DefaultHeuristics = HeuristicProfile{
	EvidenceConfidence: EvidenceConfidenceConfig{
		NoiseAdjacentConfidenceMultiplier: 0.5,
	},
	CandidateSelection: CandidateSelectionConfig{
		LargeDirectoryFileThreshold: 200,
		EvidenceDenseFileThreshold:  20,
		EvidenceDensityThreshold:    0.05,
	},
	Scoring: ScoringConfig{
		DominantExtensionLimit: 3,
		ModuleEvidenceWeight:   100,
	},
	Compression: CompressionConfig{
		CoveragePercentScale:      100,
		CompressionEvidenceWeight: 200,
		CoverageScoreWeight:       10,
		HighOverlapThreshold:      0.9,
		RedundantChildPenalty:     500,
		ChildScoreRetentionRatio:  0.6,
		NoveltyRetentionDelta:     0.2,
	},
	Diagnostics: DiagnosticsConfig{
		ExampleDirectoryLimit: 3,
	},
	RoleClassification: RoleClassificationConfig{
		FirstPartyEvidenceStrengthThreshold: 0.75,
	},
}
