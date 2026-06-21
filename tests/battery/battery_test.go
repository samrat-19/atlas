package battery

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// update rewrites the expected output files instead of comparing against them.
// Use this whenever Atlas output changes intentionally (e.g. after a Phase 1
// noise-pruning change) so the snapshots reflect the new correct behaviour.
//
// Usage:
//
//	ATLAS_BATTERY=1 go test ./tests/battery/ -update
var update = flag.Bool("update", false, "overwrite expected output files with current Atlas output")

type batteryCase struct {
	name        string
	envPath     string
	defaultPath string
	expected    string
}

func TestBatteryTextOutput(t *testing.T) {
	if os.Getenv("ATLAS_BATTERY") != "1" {
		t.Skip("set ATLAS_BATTERY=1 to run large repository battery tests")
	}

	projectRoot := findProjectRoot(t)
	binary := buildAtlas(t, projectRoot)

	cases := []batteryCase{
		{
			name:        "selenium",
			envPath:     "ATLAS_BATTERY_SELENIUM",
			defaultPath: `D:\SampleTestProjects\selenium-trunk`,
			expected:    filepath.Join(projectRoot, "tests", "battery", "resources", "selenium.expected.txt"),
		},
		{
			name:        "tensorflow",
			envPath:     "ATLAS_BATTERY_TENSORFLOW",
			defaultPath: `D:\SampleTestProjects\tensorflow-master`,
			expected:    filepath.Join(projectRoot, "tests", "battery", "resources", "tensorflow.expected.txt"),
		},
		{
			name:        "vscode",
			envPath:     "ATLAS_BATTERY_VSCODE",
			defaultPath: `D:\SampleTestProjects\vscode-main`,
			expected:    filepath.Join(projectRoot, "tests", "battery", "resources", "vscode.expected.txt"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoPath := os.Getenv(tc.envPath)
			if repoPath == "" {
				repoPath = tc.defaultPath
			}
			if _, err := os.Stat(repoPath); err != nil {
				t.Skipf("repository path unavailable: %s: %v", repoPath, err)
			}

			actual := runAtlas(t, binary, repoPath)

			if *update {
				// Normalise before writing so the file never contains a
				// machine-specific root path or Windows line endings.
				if err := os.WriteFile(tc.expected, []byte(actual), 0644); err != nil {
					t.Fatalf("failed to update expected file %s: %v", tc.expected, err)
				}
				t.Logf("updated %s", tc.expected)
				return
			}

			expectedBytes, err := os.ReadFile(tc.expected)
			if err != nil {
				t.Skipf("expected resource unavailable: %s: %v", tc.expected, err)
			}

			expected := normalizeOutput(string(expectedBytes))
			if actual != expected {
				t.Fatalf("battery output changed for %s\n\nExpected resource: %s\n\nFirst difference:\n%s",
					tc.name,
					tc.expected,
					firstDifference(expected, actual),
				)
			}
		})
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test file")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("failed to locate project root containing go.mod")
		}
		dir = parent
	}
}

func buildAtlas(t *testing.T, projectRoot string) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), executableName("atlas"))
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/atlas")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	return binary
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func runAtlas(t *testing.T, binary, repoPath string) string {
	t.Helper()

	cmd := exec.Command(binary, repoPath)
	cmd.Dir = t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("atlas failed: %v\nstderr:\n%s", err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("atlas wrote unexpected stderr:\n%s", stderr.String())
	}
	return normalizeOutput(stdout.String())
}

func normalizeOutput(output string) string {
	output = strings.TrimPrefix(output, "\ufeff")
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "Root path: ") {
			lines[i] = "Root path: <ROOT>"
		}
	}
	return strings.Join(lines, "\n")
}

func firstDifference(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	max := len(expectedLines)
	if len(actualLines) > max {
		max = len(actualLines)
	}

	for i := 0; i < max; i++ {
		var expectedLine, actualLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		} else {
			expectedLine = "<missing>"
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		} else {
			actualLine = "<missing>"
		}
		if expectedLine != actualLine {
			return "line " + itoa(i+1) + "\nexpected: " + expectedLine + "\nactual:   " + actualLine
		}
	}
	return "no textual difference found"
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}
