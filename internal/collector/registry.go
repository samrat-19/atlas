package collector

import (
	"path/filepath"
	"runtime"
	"strings"
)

// EvidenceRegistry maps evidence keys (filenames or relative paths) to categories.
// It is configurable via SetRegistry. By default it contains common filenames.
// EvidenceRegistry holds the raw registry mapping as provided by callers.
var EvidenceRegistry = defaultEvidenceRegistry()

// filenameIndex contains registry entries keyed by plain filename.
// suffixIndex contains registry entries keyed by normalized relative-path suffix.
var filenameIndex map[string]string
var suffixIndex map[string]string

func init() {
	buildIndexes(EvidenceRegistry)
}

func defaultEvidenceRegistry() map[string]string {
	return map[string]string{
		// Build systems
		"pom.xml":             "build system",
		"build.gradle":        "build system",
		"build.gradle.kts":    "build system",
		"settings.gradle":     "build system",
		"settings.gradle.kts": "build system",
		"Android.bp":          "build system",
		"Android.mk":          "build system",
		"BUILD":               "build system",
		"BUCK":                "build system",
		"meson.build":         "build system",
		"SConstruct":          "build system",
		"configure.ac":        "build system",
		"Makefile.am":         "build system",
		"BUILD.bazel":         "build system",
		"WORKSPACE":           "build system",
		"WORKSPACE.bazel":     "build system",
		"CMakeLists.txt":      "build system",
		"Makefile":            "build system",
		"build.xml":           "build system",

		// Package managers
		"package.json":       "package manager",
		"package-lock.json":  "package manager",
		"yarn.lock":          "package manager",
		"pnpm-lock.yaml":     "package manager",
		"go.mod":             "package manager",
		"go.sum":             "package manager",
		"Cargo.toml":         "package manager",
		"Cargo.lock":         "package manager",
		"requirements.txt":   "package manager",
		"pyproject.toml":     "package manager",
		"setup.py":           "package manager",
		"Pipfile":            "package manager",
		"Pipfile.lock":       "package manager",
		"poetry.lock":        "package manager",
		"poetry.toml":        "package manager",
		"environment.yml":    "package manager",
		"conda.yml":          "package manager",
		"Gemfile":            "package manager",
		"Gemfile.lock":       "package manager",
		"composer.json":      "package manager",
		"composer.lock":      "package manager",
		"packages.config":    "package manager",
		"paket.dependencies": "package manager",
		"project.clj":        "package manager",
		"mix.exs":            "package manager",
		"stack.yaml":         "package manager",
		"cabal.project":      "package manager",
		"DESCRIPTION":        "package manager",
		"Project.toml":       "package manager",

		// CI/CD
		"Jenkinsfile":             "ci/cd",
		".gitlab-ci.yml":          "ci/cd",
		".github/workflows":       "ci/cd",
		".travis.yml":             "ci/cd",
		".circleci/config.yml":    "ci/cd",
		"azure-pipelines.yml":     "ci/cd",
		"bitbucket-pipelines.yml": "ci/cd",
		".drone.yml":              "ci/cd",
		"buildkite.yml":           "ci/cd",
		"appveyor.yml":            "ci/cd",

		// Containers
		"Dockerfile":         "container",
		"docker-compose.yml": "container",
		"Containerfile":      "container",
		"Chart.yaml":         "helm",
		"kustomization.yaml": "kubernetes",
		"k8s/":               "kubernetes",
		"manifests/":         "kubernetes",
		"charts/":            "helm",
		"main.tf":            "terraform",
		"terraform.tfstate":  "terraform",
		"terraform.lock.hcl": "terraform",
		"Vagrantfile":        "vm",
		"Packerfile":         "packer",
	}
}

// SetRegistry replaces the global evidence registry. Pass nil to restore defaults.
func SetRegistry(reg map[string]string) {
	if reg == nil {
		EvidenceRegistry = defaultEvidenceRegistry()
		buildIndexes(EvidenceRegistry)
		return
	}
	EvidenceRegistry = copyRegistry(reg)
	buildIndexes(EvidenceRegistry)
}

func copyRegistry(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// buildIndexes precomputes filename and suffix maps for faster/more correct
// matching. It also applies platform-specific normalization (Windows
// case-insensitivity).
func buildIndexes(reg map[string]string) {
	filenameIndex = make(map[string]string)
	suffixIndex = make(map[string]string)
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
// category, and whether a match was found.
func MatchEvidence(name, relPath string) (key string, category string, ok bool) {
	isWindows := runtime.GOOS == "windows"

	normName := name
	if isWindows {
		normName = strings.ToLower(normName)
	}
	// filename exact match
	if cat, found := filenameIndex[normName]; found {
		return name, cat, true
	}

	// path-suffix match
	normRel := normalizeKey(relPath)
	if isWindows {
		normRel = strings.ToLower(normRel)
	}
	for suff, cat := range suffixIndex {
		if strings.HasSuffix(normRel, suff) {
			return suff, cat, true
		}
	}
	return "", "", false
}

// normalizeKey converts a registry key to a platform-independent form for
// comparison. We use forward slashes for matching path suffixes.
func normalizeKey(k string) string {
	return filepath.ToSlash(k)
}
