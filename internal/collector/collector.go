package collector

// package collector walks a repository and produces a structural inventory.
//
// Shared data shapes (Result, ModuleCandidate, HeuristicProfile, Role, and
// the static pattern data role classification reads) live in
// atlas/internal/model, which this package imports but never imports back —
// see internal/model's package doc. This package supplies all the logic:
// evidence matching (registry.go), scoring and compression (modules.go,
// module_scoring.go, module_compression.go), role classification (role.go),
// hierarchy construction (hierarchy.go, hierarchy_aggregation.go), the
// unrecognized-extension diagnostic (unrecognized.go), and orchestration
// (collect.go).
