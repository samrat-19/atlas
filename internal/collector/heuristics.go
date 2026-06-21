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
// turn a large directory tree into a shorter orientation report.

// Candidate selection constants answer: "Should Atlas consider this folder at all?"
const (
	// A folder with this many files is worth considering even if it has no
	// obvious build file. Example: a large src/ tree may matter even without its
	// own package.json or BUILD file.
	largeDirectoryFileThreshold = 200

	// We only use the "evidence density" rule for folders with at least this many
	// files. This avoids overreacting to tiny folders where 1 marker out of 1 file
	// would look like "100% evidence".
	evidenceDenseFileThreshold = 20

	// If at least this share of a folder's files are evidence files, the folder is
	// probably structurally interesting. 0.05 means 5%, so 1 evidence file in 20
	// files is enough.
	evidenceDensityThreshold = 0.05
)

// Candidate fingerprint and scoring constants answer:
// "How strong does this candidate look?"
const (
	// For each candidate folder, remember only this many common extensions. This
	// gives us a small fingerprint like [".ts", ".json", ".md"] instead of the
	// full file list.
	dominantExtensionLimit = 3

	// First-pass ranking: one evidence file counts like this many normal files.
	// This makes a folder with package/build files rank above a folder that is
	// only slightly larger.
	moduleEvidenceWeight = 100
)

// Compression constants answer:
// "When Atlas shortens the candidate list, what should survive?"
const (
	// Converts "files in this subtree / files in repo" into a percentage. Example:
	// 250 files out of 1000 becomes 25.
	coveragePercentScale = 100

	// Compression ranking: evidence matters even more here because we are deciding
	// which candidate folders survive into the shorter "major modules" list.
	compressionEvidenceWeight = 200

	// During compression, each 1% of repo coverage adds this many points. This
	// gives large subtrees some credit without letting size be the only signal.
	coverageScoreWeight = 10

	// If a child folder is this similar to its parent, Atlas treats it as probably
	// redundant. Example: parent and child are both mostly .java and both only
	// have build-system evidence.
	highOverlapThreshold = 0.9

	// Score removed from a child folder when it looks almost the same as its
	// parent. This nudges Atlas toward keeping the parent instead of every nested
	// folder with the same shape.
	redundantChildPenalty = 500

	// Keep a child folder if its score is at least this fraction of its parent's
	// score. 0.6 means a child that is 60% as strong as the parent can survive.
	childScoreRetentionRatio = 0.6

	// Keep a child folder if it is different enough from its parent. 0.2 means
	// "more than 20% different" by extension mix or evidence category.
	noveltyRetentionDelta = 0.2
)
