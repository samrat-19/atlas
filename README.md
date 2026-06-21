# Atlas

Atlas is a repository understanding engine.

It helps answer the first question engineers face in an unfamiliar repository:

> What am I looking at?

Atlas does not parse source code, resolve dependencies, generate SBOMs, or scan
for vulnerabilities. It reads repository structure and common project evidence
files to produce a quick orientation report.

## What Atlas Reports

Atlas currently reports:

- total file count
- evidence files such as `go.mod`, `package.json`, `BUILD`, `Dockerfile`, and CI files
- evidence counts by category and filename
- top evidence clusters
- top directories
- top file extensions
- inferred repository hierarchy
- candidate major modules

The current module and hierarchy logic is heuristic. It is useful for orientation,
but it is not yet a final importance model.

## Requirements

- Go 1.20 or newer

## Build

From the repository root:

```powershell
go build -o bin\atlas.exe .\cmd\atlas
```

On Unix-like systems:

```bash
go build -o bin/atlas ./cmd/atlas
```

## Run

Scan a repository:

```powershell
.\bin\atlas.exe D:\path\to\repository
```

If no path is provided, Atlas scans the current working directory:

```powershell
.\bin\atlas.exe
```

Atlas prints a text report to stdout and also writes reports under `output/`:

```text
output/
|-- <root>-<timestamp>.txt
`-- <root>-<timestamp>.json
```

## Test

Run the normal test suite:

```powershell
go test ./...
```

Run static checks:

```powershell
go vet ./...
```

## Battery Tests

Battery tests compare Atlas output against expected output for large real
repositories. They are opt-in because the repositories are external to this
workspace.

Default expected local paths:

```text
D:\SampleTestProjects\selenium-trunk
D:\SampleTestProjects\tensorflow-master
D:\SampleTestProjects\vscode-main
```

Run battery tests:

```powershell
$env:ATLAS_BATTERY = "1"
go test ./tests/battery
```

Override repository paths:

```powershell
$env:ATLAS_BATTERY_SELENIUM = "D:\path\to\selenium"
$env:ATLAS_BATTERY_TENSORFLOW = "D:\path\to\tensorflow"
$env:ATLAS_BATTERY_VSCODE = "D:\path\to\vscode"
$env:ATLAS_BATTERY = "1"
go test ./tests/battery
```

Battery tests currently compare normalized text output only. The root path is
normalized to `Root path: <ROOT>`.

## Project Layout

```text
cmd/atlas/              CLI and report rendering
internal/collector/     repository traversal, evidence aggregation, module heuristics
tests/battery/          opt-in large repository regression tests
codexPlans/             planning documents and decision log
```

## Current Phase

Atlas is in Phase 1: deterministic foundation.

The current focus is making output stable, testable, and regression-protected
before improving repository importance and role classification.
