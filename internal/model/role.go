package model

// Role is Atlas's best guess at what kind of thing a module candidate is,
// using only directory-name patterns and the confidence/evidence signals
// Phase 2 D1-D3 already established — never AI, never a content read. See
// collector.classifyRole, the only place that constructs a Role value.
type Role string

const (
	// RoleFirstParty: looks like genuine first-party project structure —
	// confident evidence, no vendored/generated/build-output/test-fixture
	// pattern match.
	RoleFirstParty Role = "first-party"

	// RoleVendored: path matches a vendored/third-party convention (see
	// VendoredPathSegments).
	RoleVendored Role = "vendored"

	// RoleGenerated: path matches a generated-code convention (see
	// GeneratedPathSegments).
	RoleGenerated Role = "generated"

	// RoleTestFixture: path matches a test/fixture/example/mock convention
	// (see NoiseAdjacentPathSegments) — the same signal D1 uses to discount
	// evidence confidence.
	RoleTestFixture Role = "test-fixture"

	// RoleBuildOutput: path matches a build-output convention (see
	// BuildOutputPathSegments).
	RoleBuildOutput Role = "build-output"

	// RoleAmbiguous: no path pattern matched, and either there is no
	// evidence at all or the evidence isn't strong enough to commit to
	// first-party. This is a legitimate, honest answer, not a fallback to
	// be avoided — see docs/heuristics.md and the determinism discussion in
	// docs/phase-2-plan.md D4.
	RoleAmbiguous Role = "ambiguous"
)
