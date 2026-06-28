package collector

import "atlas/internal/model"

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
// evidenceCount and evidenceStrength come from the same candidate. Called
// from collect.go's orchestration after buildModuleSummary, not from
// modules.go itself — scoring and classification are independent concerns,
// tied together only by Collect's sequencing.
//
// This function changes no scoring, retention, or compression behavior.
// Whether a role should ever influence retention is a separate, later
// decision (docs/phase-2-plan.md D4b), deliberately not made here.
func classifyRole(path string, evidenceCount int, evidenceStrength float64, profile model.HeuristicProfile) model.Role {
	switch {
	case model.PathContainsSegment(path, model.VendoredPathSegments, true):
		return model.RoleVendored
	case model.PathContainsSegment(path, model.GeneratedPathSegments, true):
		return model.RoleGenerated
	case model.PathContainsSegment(path, model.BuildOutputPathSegments, true):
		return model.RoleBuildOutput
	case model.PathContainsSegment(path, model.NoiseAdjacentPathSegments, true):
		return model.RoleTestFixture
	case evidenceCount > 0 && evidenceStrength >= profile.RoleClassification.FirstPartyEvidenceStrengthThreshold:
		return model.RoleFirstParty
	default:
		return model.RoleAmbiguous
	}
}
