package collector

// prunedDirectories maps exact directory names to a short policy label.
//
// When the walker encounters a directory whose name is in this map it records
// it in Result.PrunedPaths and returns fs.SkipDir — neither the directory
// itself nor any descendant is counted, matched for evidence, or considered
// as a module candidate.
//
// Only high-confidence, unambiguous names live here. A name that means
// "source" in one ecosystem and "output" in another (e.g. "build", "dist")
// stays out until a smarter classification exists. Being wrong about a
// genuine source directory is worse than including a little noise.
var prunedDirectories = map[string]string{
	// .git holds version-control internals — object store, refs, hooks, and
	// config files. Several of its paths match evidence rules by accident:
	// .git/description would fire the R-package DESCRIPTION rule, and
	// .git/refs/heads looks like a path-suffix match for some registries.
	// None of it is source code.
	".git": "version-control",

	// node_modules contains every package installed by npm/yarn/pnpm.
	// A single JS project can hold tens of thousands of files here, each
	// with its own package.json. Including it would make every transitive
	// dependency look like a first-party module and drown the real signal.
	"node_modules": "installed-dependencies",

	// bazel-out is Bazel's build output tree. It mirrors the full source
	// layout and is often larger than the source itself. Traversing it would
	// double-count every source region and promote build artifacts into the
	// module candidate list.
	"bazel-out": "build-output",

	// .terraform caches provider plugins downloaded by terraform init.
	// The plugins are large binaries; the actual Terraform HCL source lives
	// alongside .tf files in the project directories, not here.
	".terraform": "tool-cache",

	// __pycache__ holds CPython bytecode (.pyc/.pyo) compiled from adjacent
	// .py files. It mirrors the source layout exactly and adds no structural
	// information beyond what the source already provides.
	"__pycache__": "build-output",

	// .cache is a catch-all written by many tools: Gradle, pip, Cargo,
	// Maven, and others. Contents are always tool-managed, never hand-written
	// source.
	".cache": "tool-cache",
}

// prunePolicy returns the policy label for a directory that should be skipped,
// and reports whether the name is in the prune list. The caller is responsible
// for not pruning the scan root itself.
func prunePolicy(name string) (policy string, ok bool) {
	policy, ok = prunedDirectories[name]
	return
}
