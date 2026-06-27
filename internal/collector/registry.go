package collector

import (
	"path/filepath"
	"runtime"
	"strings"
)

// EvidenceRule describes one entry in the evidence registry: the category it
// implies and how strong a signal it is on its own.
type EvidenceRule struct {
	// Category is the evidence classification (e.g. "build system",
	// "package manager") used throughout topology, cluster, and module
	// summaries.
	Category string

	// Confidence is this rule's intrinsic signal strength in isolation, from
	// 0 (no signal) to 1 (unambiguous). The registry key a rule is stored
	// under (filename or path suffix) is its stable identity; there is no
	// separate ID field because nothing today needs to reference a rule
	// independently of how it is matched.
	Confidence float64
}

// EvidenceRegistry maps evidence keys (filenames or relative-path suffixes)
// to typed rules. It is configurable via SetRegistry. By default it contains
// common filenames, each starting at full confidence — see
// pathContextMultiplier for the one contextual adjustment Atlas currently
// applies (matches found under test/fixture/example-style directories).
var EvidenceRegistry = defaultEvidenceRegistry()

// filenameIndex contains registry entries keyed by plain filename.
// suffixIndex contains registry entries keyed by normalized relative-path suffix.
var filenameIndex map[string]EvidenceRule
var suffixIndex map[string]EvidenceRule

func init() {
	buildIndexes(EvidenceRegistry)
}

func defaultEvidenceRegistry() map[string]EvidenceRule {
	return map[string]EvidenceRule{
		// Build systems
		"pom.xml":             {Category: "build system", Confidence: 1.0},
		"build.gradle":        {Category: "build system", Confidence: 1.0},
		"build.gradle.kts":    {Category: "build system", Confidence: 1.0},
		"settings.gradle":     {Category: "build system", Confidence: 1.0},
		"settings.gradle.kts": {Category: "build system", Confidence: 1.0},
		"Android.bp":          {Category: "build system", Confidence: 1.0},
		"Android.mk":          {Category: "build system", Confidence: 1.0},
		"BUILD":               {Category: "build system", Confidence: 1.0},
		"BUCK":                {Category: "build system", Confidence: 1.0},
		"meson.build":         {Category: "build system", Confidence: 1.0},
		"SConstruct":          {Category: "build system", Confidence: 1.0},
		"configure.ac":        {Category: "build system", Confidence: 1.0},
		"Makefile.am":         {Category: "build system", Confidence: 1.0},
		"BUILD.bazel":         {Category: "build system", Confidence: 1.0},
		"WORKSPACE":           {Category: "build system", Confidence: 1.0},
		"WORKSPACE.bazel":     {Category: "build system", Confidence: 1.0},
		"CMakeLists.txt":      {Category: "build system", Confidence: 1.0},
		"Makefile":            {Category: "build system", Confidence: 1.0},
		"build.xml":           {Category: "build system", Confidence: 1.0},

		// Package managers
		"package.json":       {Category: "package manager", Confidence: 1.0},
		"package-lock.json":  {Category: "package manager", Confidence: 1.0},
		"yarn.lock":          {Category: "package manager", Confidence: 1.0},
		"pnpm-lock.yaml":     {Category: "package manager", Confidence: 1.0},
		"go.mod":             {Category: "package manager", Confidence: 1.0},
		"go.sum":             {Category: "package manager", Confidence: 1.0},
		"Cargo.toml":         {Category: "package manager", Confidence: 1.0},
		"Cargo.lock":         {Category: "package manager", Confidence: 1.0},
		"requirements.txt":   {Category: "package manager", Confidence: 1.0},
		"pyproject.toml":     {Category: "package manager", Confidence: 1.0},
		"setup.py":           {Category: "package manager", Confidence: 1.0},
		"Pipfile":            {Category: "package manager", Confidence: 1.0},
		"Pipfile.lock":       {Category: "package manager", Confidence: 1.0},
		"poetry.lock":        {Category: "package manager", Confidence: 1.0},
		"poetry.toml":        {Category: "package manager", Confidence: 1.0},
		"environment.yml":    {Category: "package manager", Confidence: 1.0},
		"conda.yml":          {Category: "package manager", Confidence: 1.0},
		"Gemfile":            {Category: "package manager", Confidence: 1.0},
		"Gemfile.lock":       {Category: "package manager", Confidence: 1.0},
		"composer.json":      {Category: "package manager", Confidence: 1.0},
		"composer.lock":      {Category: "package manager", Confidence: 1.0},
		"packages.config":    {Category: "package manager", Confidence: 1.0},
		"paket.dependencies": {Category: "package manager", Confidence: 1.0},
		"project.clj":        {Category: "package manager", Confidence: 1.0},
		"mix.exs":            {Category: "package manager", Confidence: 1.0},
		"stack.yaml":         {Category: "package manager", Confidence: 1.0},
		"cabal.project":      {Category: "package manager", Confidence: 1.0},
		"DESCRIPTION":        {Category: "package manager", Confidence: 1.0},
		"Project.toml":       {Category: "package manager", Confidence: 1.0},

		// CI/CD
		"Jenkinsfile":             {Category: "ci/cd", Confidence: 1.0},
		".gitlab-ci.yml":          {Category: "ci/cd", Confidence: 1.0},
		".github/workflows":       {Category: "ci/cd", Confidence: 1.0},
		".travis.yml":             {Category: "ci/cd", Confidence: 1.0},
		".circleci/config.yml":    {Category: "ci/cd", Confidence: 1.0},
		"azure-pipelines.yml":     {Category: "ci/cd", Confidence: 1.0},
		"bitbucket-pipelines.yml": {Category: "ci/cd", Confidence: 1.0},
		".drone.yml":              {Category: "ci/cd", Confidence: 1.0},
		"buildkite.yml":           {Category: "ci/cd", Confidence: 1.0},
		"appveyor.yml":            {Category: "ci/cd", Confidence: 1.0},

		// Containers
		"Dockerfile":         {Category: "container", Confidence: 1.0},
		"docker-compose.yml": {Category: "container", Confidence: 1.0},
		"Containerfile":      {Category: "container", Confidence: 1.0},
		"Chart.yaml":         {Category: "helm", Confidence: 1.0},
		"kustomization.yaml": {Category: "kubernetes", Confidence: 1.0},
		"k8s/":               {Category: "kubernetes", Confidence: 1.0},
		"manifests/":         {Category: "kubernetes", Confidence: 1.0},
		"charts/":            {Category: "helm", Confidence: 1.0},
		"main.tf":            {Category: "terraform", Confidence: 1.0},
		"terraform.tfstate":  {Category: "terraform", Confidence: 1.0},
		"terraform.lock.hcl": {Category: "terraform", Confidence: 1.0},
		"Vagrantfile":        {Category: "vm", Confidence: 1.0},
		"Packerfile":         {Category: "packer", Confidence: 1.0},
	}
}

// noiseAdjacentPathSegments names directory components that conventionally
// hold test, fixture, example, or mock content rather than first-party
// project structure. A match found beneath one of these is still recorded as
// evidence — Atlas cannot rule out that it is genuine — but with reduced
// confidence. See pathContextMultiplier.
var noiseAdjacentPathSegments = map[string]struct{}{
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

// pathContextMultiplier discounts a rule's intrinsic confidence when the
// matched evidence sits under a noise-adjacent directory (see
// noiseAdjacentPathSegments). The final path segment is the matched file or
// path-suffix itself, not a containing directory, so it is excluded from the
// check. The discount amount comes from profile rather than a bare constant
// so a future profile could tune or disable it.
func pathContextMultiplier(relPath string, profile HeuristicProfile) float64 {
	segments := strings.Split(filepath.ToSlash(relPath), "/")
	for _, segment := range segments[:len(segments)-1] {
		if _, ok := noiseAdjacentPathSegments[strings.ToLower(segment)]; ok {
			return profile.EvidenceConfidence.NoiseAdjacentConfidenceMultiplier
		}
	}
	return 1.0
}

// SetRegistry replaces the global evidence registry. Pass nil to restore defaults.
func SetRegistry(reg map[string]EvidenceRule) {
	if reg == nil {
		EvidenceRegistry = defaultEvidenceRegistry()
		buildIndexes(EvidenceRegistry)
		return
	}
	EvidenceRegistry = copyRegistry(reg)
	buildIndexes(EvidenceRegistry)
}

func copyRegistry(src map[string]EvidenceRule) map[string]EvidenceRule {
	dst := make(map[string]EvidenceRule, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// buildIndexes precomputes filename and suffix maps for faster/more correct
// matching. It also applies platform-specific normalization (Windows
// case-insensitivity).
func buildIndexes(reg map[string]EvidenceRule) {
	filenameIndex = make(map[string]EvidenceRule)
	suffixIndex = make(map[string]EvidenceRule)
	isWindows := runtime.GOOS == "windows"

	for k, v := range reg {
		norm := normalizeKey(k)
		if isWindows {
			norm = strings.ToLower(norm)
		}
		if strings.Contains(norm, "/") {
			suffixIndex[norm] = v
			continue
		}
		filenameIndex[norm] = v
	}
}

// MatchEvidence attempts to match an entry by filename first, then by
// relative-path suffix. It returns the registry key (as provided), the
// category, a confidence score adjusted for path context using profile, and
// whether a match was found.
func MatchEvidence(name, relPath string, profile HeuristicProfile) (key string, category string, confidence float64, ok bool) {
	isWindows := runtime.GOOS == "windows"

	normName := name
	if isWindows {
		normName = strings.ToLower(normName)
	}
	// filename exact match
	if rule, found := filenameIndex[normName]; found {
		return name, rule.Category, rule.Confidence * pathContextMultiplier(relPath, profile), true
	}

	// path-suffix match
	normRel := normalizeKey(relPath)
	if isWindows {
		normRel = strings.ToLower(normRel)
	}
	for suff, rule := range suffixIndex {
		if strings.HasSuffix(normRel, suff) {
			return suff, rule.Category, rule.Confidence * pathContextMultiplier(relPath, profile), true
		}
	}
	return "", "", 0, false
}

// normalizeKey converts a registry key to a platform-independent form for
// comparison. We use forward slashes for matching path suffixes.
func normalizeKey(k string) string {
	return filepath.ToSlash(k)
}
