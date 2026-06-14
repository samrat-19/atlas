package collector

// Types used by the collector package.

// Result is returned by Collect and contains the root path and total file count
// and discovered evidence.
type Result struct {
	// Root is the provided directory root path that was analyzed.
	Root string

	// TotalFiles is the number of files (not directories) found under Root.
	TotalFiles int

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
}

// EvidenceItem describes a single discovered evidence entry.
type EvidenceItem struct {
	Filename     string
	AbsolutePath string
	RelativePath string
	Category     string
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
type ModuleCandidate struct {
	Path               string
	FileCount          int
	EvidenceCount      int
	DominantExtensions []string
	EvidenceByCategory map[string]int
	EvidenceByFilename map[string]int
	Score              int
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
