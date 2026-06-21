# Atlas Battery Tests

Battery tests compare Atlas output against expected output for large, real
repositories. They are opt-in because the repositories are external to this
workspace and can be expensive to scan.

## Repositories

The default local paths are:

- Selenium: `D:\SampleTestProjects\selenium-trunk`
- TensorFlow: `D:\SampleTestProjects\tensorflow-master`
- VS Code: `D:\SampleTestProjects\vscode-main`

Override them with:

```powershell
$env:ATLAS_BATTERY_SELENIUM = "D:\path\to\selenium"
$env:ATLAS_BATTERY_TENSORFLOW = "D:\path\to\tensorflow"
$env:ATLAS_BATTERY_VSCODE = "D:\path\to\vscode"
```

## Running

```powershell
$env:ATLAS_BATTERY = "1"
go test ./tests/battery
```

Without `ATLAS_BATTERY=1`, the tests are skipped.

## Normalization

Battery tests currently compare text output only. They normalize:

- line endings
- the `Root path: ...` line, replacing the absolute local path with
  `Root path: <ROOT>`

JSON is intentionally not compared yet because current JSON includes absolute
paths. Canonical JSON should be added after Phase 1 introduces a stable snapshot
format.
