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
go test ./tests/battery/
```

Without `ATLAS_BATTERY=1`, the tests are skipped.

## Updating expected output

When Atlas output changes intentionally (e.g. after a noise-pruning fix or a
new report section), the expected snapshot files need to be updated. Use the
`-update` flag to regenerate them in one step:

```powershell
$env:ATLAS_BATTERY = "1"
go test ./tests/battery/ -update
```

This runs Atlas against each repository, normalizes the output, and overwrites
the corresponding `resources/*.expected.txt` file. Review the diff before
committing — the updated files are the new correctness baseline.

## Normalization

Battery tests normalize output before comparing or writing:

- Windows line endings (`\r\n`) are converted to `\n`
- The `Root path: ...` line is replaced with `Root path: <ROOT>` so the
  snapshot is not machine-specific

JSON is intentionally not compared because it includes absolute paths and
timestamps. Canonical JSON comparison should be added after a stable snapshot
format is introduced.
