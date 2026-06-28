# Atlas

Atlas is a repository understanding engine.

It helps answer the first question engineers face in an unfamiliar repository:

> What am I looking at?

Atlas does not parse source code, resolve dependencies, generate SBOMs, or scan
for vulnerabilities. It reads structure — file distribution, build/package/CI/IaC
manifests, directory shape — and turns that into an orientation report.

That boundary is deliberate, not a missing feature: Atlas never reads file
contents, so its output is safe to share with a teammate, a support ticket, or
an auditor without exposing a single line of source code.

## What you get

- Total file count, and which top-level directories and extensions dominate the repository.
- **Evidence**: build files, package managers, CI/CD configs, and container/IaC manifests Atlas
  recognizes (`go.mod`, `package.json`, `BUILD`, `Dockerfile`, CI configs, and more) — each with a
  confidence score, not just a yes/no. Evidence found under a `test/`, `fixtures/`, or `examples/`
  style path is recorded with reduced confidence rather than treated the same as a real project root.
- **Major modules**, each with five separate, named numbers instead of one opaque importance score:
  *boundary confidence*, *evidence strength*, *structural prominence*, *novelty vs. its parent*, and
  *noise probability*.
- **A structural role per module** — `first-party`, `vendored`, `generated`, `test-fixture`,
  `build-output`, or `ambiguous`. When the structure genuinely doesn't support a confident answer,
  Atlas says `ambiguous` rather than guessing.
- **An inferred repository hierarchy** (regions → subsystems → components) built from the retained modules.
- **A diagnostic for what Atlas couldn't explain** — large, evidence-less directories grouped by
  shared file extension, so a recurring unrecognized pattern is visible instead of silently ignored.

All of it is deterministic. The same repository always produces the same report, byte for byte —
no AI, no network calls, no randomness.

## Example

A few entries from a real run against a large open-source repository:

```text
- third_party/xla/xla/service
  score: 707
  files: 477
  evidence: 1
  boundary confidence: 0.67
  evidence strength: 1.00
  structural prominence: 0.03
  novelty vs parent: 0.33
  noise probability: 0.00
  role: vendored

- tensorflow/tools/android/test
  score: 610
  files: 10
  evidence: 3
  boundary confidence: 0.75
  evidence strength: 0.50
  structural prominence: 0.00
  novelty vs parent: 1.00
  noise probability: 0.50
  role: test-fixture

- tensorflow/tools/api/golden/v2
  score: 609
  files: 599
  evidence: 0
  boundary confidence: 0.50
  evidence strength: 0.00
  structural prominence: 0.01
  novelty vs parent: 1.00
  noise probability: 0.50
  role: ambiguous
```

`third_party/xla` is correctly labeled `vendored` from its path alone. The `test/` directory is
labeled `test-fixture`, and its evidence confidence is visibly halved rather than treated as a real
project root. `tools/api/golden/v2` has no recognizable evidence at all — 599 files Atlas genuinely
can't explain — so it's reported as `ambiguous` instead of a confident guess.

Atlas also flags when an unrecognized pattern recurs, instead of staying silent about it:

```text
Unrecognized Extension Clusters:
Total unrecognized directories: 1
Total unrecognized files: 504

- .ts (1 directories, 504 files)
  examples:
  - src/vs/platform/agentHost/node/codex/protocol/generated/v2
```

That path contains "generated" — a real, observed gap that fed directly back into Atlas's
classification rules.

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

Atlas prints a text report to stdout and also writes both a text and JSON snapshot under `output/`:

```text
output/
|-- <root>-<timestamp>.txt
`-- <root>-<timestamp>.json
```

The JSON snapshot carries a `SchemaVersion` field so downstream consumers can detect breaking
changes to its shape.

## Development

```powershell
go test ./...
go vet ./...
```

See `CLAUDE.md` for architecture (the `internal/model` / `internal/collector` package split, the
evidence/scoring/classification pipeline) and `tests/battery/README.md` for the opt-in regression
suite that runs Atlas against large real-world repositories.
