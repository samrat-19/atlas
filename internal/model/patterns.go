package model

import (
	"path/filepath"
	"strings"
)

// Directory-name patterns that hint at a candidate's structural role,
// independent of any evidence file. Each set below is deliberately small
// and conservative — a pattern only belongs here once it is a
// well-established, low-ambiguity convention. This is not meant to be
// exhaustive; see collector/unrecognized.go's diagnostic for how gaps get
// found from real repository structure instead of being guessed upfront.
//
// These live in model, not paired with the evidence registry, because they
// are consumed by two otherwise-independent concerns — evidence confidence
// discounting (collector.pathContextMultiplier) and role classification
// (collector.classifyRole) — that should not depend on each other. Putting
// the data in the shared base package lets both import it without either
// importing the other.

// NoiseAdjacentPathSegments names directory components that conventionally
// hold test, fixture, example, or mock content rather than first-party
// project structure. Used both to discount evidence confidence (Phase 2 D1)
// and to classify a candidate's structural role as "test-fixture" (Phase 2
// D4).
var NoiseAdjacentPathSegments = map[string]struct{}{
	"test":         {},
	"tests":        {},
	"testdata":     {},
	"fixture":      {},
	"fixtures":     {},
	"example":      {},
	"examples":     {},
	"sample":       {},
	"samples":      {},
	"mock":         {},
	"mocks":        {},
	"__mocks__":    {},
	"__fixtures__": {},
	"__tests__":    {},
}

// VendoredPathSegments names directories that conventionally hold copied-in
// third-party code rather than first-party source. node_modules is not
// listed here — it is hard-pruned (see collector/prune.go) and never
// becomes a candidate at all, so there is nothing for role classification
// to see.
var VendoredPathSegments = map[string]struct{}{
	"vendor":      {},
	"third_party": {},
	"thirdparty":  {},
}

// GeneratedPathSegments names directories whose contents are produced by a
// tool rather than hand-written. Starts narrow and grows from observed
// cases, not a guessed exhaustive list: the unrecognized-extension
// diagnostic that preceded this list found a real 504-file .ts cluster at a
// VS Code path containing "generated" with zero evidence files, which is
// the case this list was built to catch.
var GeneratedPathSegments = map[string]struct{}{
	"generated":     {},
	"__generated__": {},
}

// BuildOutputPathSegments names directories that conventionally hold build
// artifacts. Deliberately narrow: common build-output names like "build",
// "bin", and "out" are excluded because they are also common *source*
// directory names in some ecosystems (a Go or Bazel build/ can be source —
// see collector/prune.go's rationale for not pruning "build" outright).
// Being wrong about a genuine source directory is worse than missing a
// build-output label.
var BuildOutputPathSegments = map[string]struct{}{
	"dist":   {},
	"target": {},
}

// PathContainsSegment reports whether any directory component of relPath
// (case-insensitive) is in segments.
//
// includeFinalSegment controls whether the last path component is checked:
// callers matching evidence files (e.g. collector.pathContextMultiplier)
// pass false, since the final segment there is the matched filename, not a
// containing directory. Callers matching a module candidate's own
// directory path (e.g. collector.classifyRole) pass true, since every
// segment of a directory path is itself a directory name.
func PathContainsSegment(relPath string, segments map[string]struct{}, includeFinalSegment bool) bool {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if !includeFinalSegment && len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	for _, part := range parts {
		if _, ok := segments[strings.ToLower(part)]; ok {
			return true
		}
	}
	return false
}
