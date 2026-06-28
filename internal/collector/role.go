package collector

// Role is Atlas's best guess at what kind of thing a module candidate is,
// using only directory-name patterns and the confidence/evidence signals
// Phase 2 D1-D3 already established — never AI, never a content read. See
// classifyRole.
type Role string

const (
	// RoleFirstParty: looks like genuine first-party project structure —
	// confident evidence, no vendored/generated/build-output/test-fixture
	// pattern match.
	RoleFirstParty Role = "first-party"

	// RoleVendored: path matches a vendored/third-party convention (see
	// vendoredPathSegments in patterns.go).
	RoleVendored Role = "vendored"

	// RoleGenerated: path matches a generated-code convention (see
	// generatedPathSegments in patterns.go).
	RoleGenerated Role = "generated"

	// RoleTestFixture: path matches a test/fixture/example/mock convention
	// (see noiseAdjacentPathSegments in patterns.go) — the same signal D1
	// uses to discount evidence confidence.
	RoleTestFixture Role = "test-fixture"

	// RoleBuildOutput: path matches a build-output convention (see
	// buildOutputPathSegments in patterns.go).
	RoleBuildOutput Role = "build-output"

	// RoleAmbiguous: no path pattern matched, and either there is no
	// evidence at all or the evidence isn't strong enough to commit to
	// first-party. This is a legitimate, honest answer, not a fallback to
	// be avoided — see docs/heuristics.md and the determinism discussion in
	// docs/phase-2-plan.md D4.
	RoleAmbiguous Role = "ambiguous"
)

// classifyRole assigns a structural role to a module candidate using a
// fixed, ordered list of checks: path patterns first (most specific, least
// ambiguous conventions), then the candidate's own evidence strength, then
// "ambiguous" if nothing fires confidently. The order is the entire
// determinism story here — a path that could plausibly match more than one
// pattern (e.g. a vendored directory that also happens to contain the word
// "generated") always resolves to whichever check is listed first, not
// whichever a map iteration happens to visit first. Every candidate is
// checked against the same rules in the same sequence, so the same
// repository structure always produces the same role.
//
// path is the candidate's own directory path (every segment is a directory
// name, unlike the evidence-file paths pathContextMultiplier checks).
// evidenceCount and evidenceStrength come from the same candidate — see
// newModuleCandidate in modules.go, the only caller.
//
// This function changes no scoring, retention, or compression behavior.
// Whether a role should ever influence retention is a separate, later
// decision (docs/phase-2-plan.md D4b), deliberately not made here.
func classifyRole(path string, evidenceCount int, evidenceStrength float64, profile HeuristicProfile) Role {
	switch {
	case pathContainsSegment(path, vendoredPathSegments, true):
		return RoleVendored
	case pathContainsSegment(path, generatedPathSegments, true):
		return RoleGenerated
	case pathContainsSegment(path, buildOutputPathSegments, true):
		return RoleBuildOutput
	case pathContainsSegment(path, noiseAdjacentPathSegments, true):
		return RoleTestFixture
	case evidenceCount > 0 && evidenceStrength >= profile.RoleClassification.FirstPartyEvidenceStrengthThreshold:
		return RoleFirstParty
	default:
		return RoleAmbiguous
	}
}
