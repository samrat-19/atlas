package collector

// Types used by the collector package.

// Result is returned by Collect and contains the root path and total file count
// and discovered evidence.
type Result struct {
	// SchemaVersion identifies the shape of this Result so consumers can
	// detect breaking changes. Increment when a field is removed or its
	// meaning changes; adding an optional field does not require a bump.
	SchemaVersion string

	// Root is the provided directory root path that was analyzed.
	Root string

	// TotalFiles is the number of files (not directories) found under Root.
	// Files inside pruned directories are not counted.
	TotalFiles int

	// PrunedPaths lists directories that were skipped during traversal.
	// Each entry names the path and the policy that triggered the skip.
	// Their files are excluded from TotalFiles and all statistics.
	PrunedPaths []PrunedPath

	// Evidence lists discovered evidence items found under Root.
	Evidence []EvidenceItem

	// TopologySummary contains aggregated evidence counts and location data.
	TopologySummary TopologySummary

	// ClusterSummary contains evidence counts grouped by top-level cluster.
	ClusterSummary ClusterSummary

	// CensusSummary contains file and evidence counts per top-level directory.
	CensusSummary CensusSummary

	// ExtensionSummary contains aggregated file extension counts.
	ExtensionSummary ExtensionSummary

	// ModuleSummary contains discovered module candidates.
	ModuleSummary ModuleSummary

	// CompressedModuleSummary contains a pruned set of high-value modules.
	CompressedModuleSummary CompressedModuleSummary

	// HierarchySummary contains a repository hierarchy view built from retained modules.
	HierarchySummary HierarchySummary

	// UnrecognizedSummary surfaces large, evidence-less module candidates
	// grouped by shared extension — a diagnostic for finding gaps in the
	// evidence registry, not part of scoring or retention.
	UnrecognizedSummary UnrecognizedSummary
}

// PrunedPath records a directory that was skipped during traversal.
// Its files are not counted in TotalFiles and its contents do not contribute
// to evidence, extension counts, or module candidates.
type PrunedPath struct {
	// RelativePath is the repository-relative path of the skipped directory,
	// using forward slashes regardless of the host operating system.
	RelativePath string

	// Policy is the short label of the rule that caused the skip.
	// See prunedDirectories in prune.go for the full set of labels.
	Policy string
}

// EvidenceItem describes a single discovered evidence entry.
type EvidenceItem struct {
	Filename     string
	AbsolutePath string
	RelativePath string
	Category     string

	// Confidence is the matched rule's signal strength, from 0 to 1, after
	// any path-context adjustment (see MatchEvidence). It is not yet used by
	// scoring or compression — Phase 2 D1 only establishes the data.
	Confidence float64
}

// TopologySummary contains counts for discovered evidence.
type TopologySummary struct {
	// TotalEvidenceItems is the total number of evidence entries discovered.
	TotalEvidenceItems int

	// EvidenceCountByCategory groups evidence count by category.
	EvidenceCountByCategory map[string]int

	// EvidenceCountByFilename groups evidence count by filename.
	EvidenceCountByFilename map[string]int

	// RootEvidenceCount is the number of evidence items directly under the root.
	RootEvidenceCount int

	// NestedEvidenceCount is the number of evidence items found in nested directories.
	NestedEvidenceCount int
}

// EvidenceCluster contains aggregated evidence information for a cluster.
type EvidenceCluster struct {
	ClusterName string

	EvidenceItemCount int

	// EvidenceCountByCategory groups evidence count by category within the cluster.
	EvidenceCountByCategory map[string]int

	// EvidenceCountByFilename groups evidence count by filename within the cluster.
	EvidenceCountByFilename map[string]int
}

// ClusterSummary groups evidence clusters across the repository.
type ClusterSummary struct {
	Clusters map[string]EvidenceCluster
}

// DirectoryCensus contains per top-level directory statistics.
type DirectoryCensus struct {
	DirectoryName     string
	TotalFiles        int
	EvidenceItemCount int
}

// CensusSummary collects top-level directory counts and evidence tallies.
type CensusSummary struct {
	Directories      map[string]DirectoryCensus
	TotalDirectories int
}

// ExtensionSummary holds global and per-cluster extension counts.
type ExtensionSummary struct {
	// ByExtension maps normalized extension (including leading dot, or "" for no extension)
	// to total file counts.
	ByExtension map[string]int

	// ByCluster maps cluster name -> (extension -> count).
	ByCluster map[string]map[string]int
}

// RegionNode represents a node in the repository hierarchy view.
type RegionNode struct {
	Path          string
	FileCount     int
	EvidenceCount int
	Score         int
	Children      []*RegionNode
}

// HierarchySummary contains a repository hierarchy built from retained modules.
type HierarchySummary struct {
	TotalRegions    int
	TotalSubsystems int
	Regions         []*RegionNode
}

// ModuleCandidate represents a directory that looks like an independent module
// or subsystem based on structure, file counts, evidence density, and
// extension fingerprints.
//
// Score is the single compression-stage number Atlas has always used for
// sorting and retention (see module_scoring.go). It is kept for backward
// compatibility, but it collapses several different ideas — size, evidence,
// coverage, redundancy — into one value with no way to tell them apart. The
// five fields below (Phase 2 D3) are the named, explainable breakdown of
// that same judgment, per docs/heuristics.md's stated direction:
//
//   - EvidenceStrength: average confidence of this directory's own evidence
//     (0 if it has none). Confidence is discounted for matches under
//     noise-adjacent paths — see EvidenceItem.Confidence and registry.go.
//   - NoiseProbability: how likely this candidate is incidental noise rather
//     than a real boundary. Derived from EvidenceStrength where evidence
//     exists; 0.5 ("no signal either way") when a candidate qualified
//     purely on size with zero evidence, since a large evidence-less
//     directory can be entirely legitimate (docs/heuristics.md Rule 2) —
//     Atlas should not assert noise it cannot support.
//   - StructuralProminence: how much of the repository's files live under
//     this candidate's subtree (0–1, the same repository-coverage
//     percentage Score already factors in, exposed on its own).
//   - NoveltyVsParent: how different this candidate is from its parent
//     candidate, by extension mix or evidence category (1.0 for a
//     candidate with no parent — nothing to be redundant with).
//   - BoundaryConfidence: how confident Atlas is that this is a genuine,
//     distinct module boundary rather than a redundant nested view of its
//     parent. The average of EvidenceStrength and NoveltyVsParent — strong,
//     confident evidence and/or meaningful difference from the parent both
//     support treating this as a real boundary.
//
// All five are 0–1 and computed deterministically from structure; none of
// them involve AI or a content read. See docs/phase-2-plan.md D3.
type ModuleCandidate struct {
	Path               string
	FileCount          int
	EvidenceCount      int
	DominantExtensions []string
	EvidenceByCategory map[string]int
	EvidenceByFilename map[string]int
	Score              int

	EvidenceStrength     float64
	NoiseProbability     float64
	StructuralProminence float64
	NoveltyVsParent      float64
	BoundaryConfidence   float64
}

// ModuleSummary aggregates discovered module candidates.
type ModuleSummary struct {
	TotalModules int
	Modules      []ModuleCandidate
}

// CompressedModuleSummary represents a pruned set of high-value module
// candidates after compression and scoring.
type CompressedModuleSummary struct {
	TotalCandidates    int
	RetainedCandidates int
	CompressionRatio   float64 // retained/total
	Modules            []ModuleCandidate
}

// UnrecognizedExtensionCluster groups module candidates that qualified
// purely by size (no evidence at all — see isLargeDirectory) and share a
// dominant file extension. A cluster with a high DirectoryCount means Atlas
// repeatedly found a large, unexplained directory with this extension
// signature across the repository — a concrete, data-driven signal that the
// evidence registry may be missing a rule for whatever this extension
// represents, rather than a one-off the existing heuristics already handle
// reasonably (see docs/heuristics.md's noiseProbability discussion).
type UnrecognizedExtensionCluster struct {
	Extension string

	// DirectoryCount is how many evidence-less large directories had this
	// extension among their dominant extensions. This is the primary
	// ranking signal: a pattern that recurs across many directories is a
	// stronger registry-gap signal than one large directory with many files.
	DirectoryCount int

	// TotalFiles is the sum of FileCount across those directories. This is a
	// coarse weight, not an exact per-extension file count — a directory's
	// FileCount includes every file in it, not just files of this specific
	// extension (DominantExtensions only records which extensions are
	// common, not how many files have each one).
	TotalFiles int

	// ExamplePaths lists a few of the matching directories, capped at
	// HeuristicProfile.Diagnostics.ExampleDirectoryLimit, so a human
	// reviewing this output has somewhere concrete to look. Sorted
	// alphabetically for readability.
	ExamplePaths []string
}

// UnrecognizedSummary reports module candidates Atlas could say nothing
// about beyond "this is large" — no evidence, no category, no filename it
// recognizes. This does not change any scoring or retention decision; it is
// a diagnostic for finding gaps in the evidence registry by looking at what
// large, unexplained directories actually have in common across a real
// repository, instead of guessing what patterns might be missing.
type UnrecognizedSummary struct {
	// TotalUnrecognizedDirectories is the count of module candidates with
	// zero evidence (qualified purely by size).
	TotalUnrecognizedDirectories int

	// TotalUnrecognizedFiles is the sum of FileCount across those
	// directories.
	TotalUnrecognizedFiles int

	// Clusters groups those directories by shared dominant extension, sorted
	// by DirectoryCount descending (the strongest "this keeps recurring"
	// signal first), then TotalFiles descending, then Extension ascending
	// for a full, deterministic tie-break.
	Clusters []UnrecognizedExtensionCluster
}
