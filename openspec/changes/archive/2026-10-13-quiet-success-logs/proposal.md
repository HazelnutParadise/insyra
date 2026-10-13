# Proposal: quiet-success-logs

## Why

api-review C-9 (#215) asked whether a library should log at Info when a call succeeds. Measured on 2026-10-13 on `0.4` at 4daa9af0, 26 `LogInfo` calls remain.

- **Eleven only repeat what a returned nil error already says.** They are the four `csvxl` conversion summaries (`CSVToExcel`, `AppendCSVToExcel`, `ExcelToCSV`, and `excelFileToCsv` behind `ExcelDirToCSV`), `plot.SaveHTML`, the two `plot.SavePNG` lines, `gplot.SaveChart`, and `py.PipInstall`, `PipUninstall` and `PipList`.
- **The CLI repeats them.** Its `convert` and `plot` commands print their own confirmation right after the library's, so a script run with the default `--log-level info` shows `Converted 1 of 1 CSV files to Excel file u.xlsx.` and then `converted u.csv to u.xlsx`.
- **The other fifteen say something the return value does not:**
  - an encoding guessed for a file (`ReadCSV` and `csvxl`, 3 lines);
  - `ToSQL` adding a column to an existing database table;
  - progress of a long operation: `nn.Sequential.Fit`'s epochs (2 lines), and the first Python download, setup and reinstall (5 lines);
  - the acceleration and thread-safety switches changing state (4 lines).
- **Measured on the CLI:** in a two-row Big5 CSV the encoding line was the only sign of a wrong guess.

The owner ruled on 2026-10-13 ([comment](https://github.com/HazelnutParadise/insyra/issues/215#issuecomment-6146697581)) to keep `LogLevelInfo` as the default, delete the eleven, and write the rule down so new code follows it.

## What Changes

- **Lines removed:** the eleven Info lines above. Each function returns what it returned before. The four `csvxl` functions keep their encoding-detection line, and `SavePNG` keeps its warning before the online fallback.
- **New Key Convention in `AGENTS.md`:** log at Info only what the return value does not say. That means a guess made about the data, a change made outside the call's own result, progress of a long operation, or a switch changing state. Never log a line that only confirms success.
- **`Docs/Configuration.md`:** says what insyra logs at Info.
- **Follow-up in `AGENTS.md`:** encoding detection is unstable on a very short Big5 file.

## Capabilities

### New Capabilities

- `log-content`: what insyra writes to its log at Info level.

### Modified Capabilities

None. No existing spec requires the removed lines.

## Impact

- `csvxl/convert.go`, `csvxl/convertDir.go`, `plot/save_chart.go`, `gplot/save_chart.go`, `py/py.go`, and two new tests.
- `AGENTS.md`, `Docs/Configuration.md`, the changelogs (under `### Core`, since the rule spans packages), `api-review.md` (C-9), `delivery-status.md`.
- **Who notices:** a program or script that read these lines in its log. No return value changes. The error buffer and `SetDefaultErrHandlingFunc` never received Info lines, so they see no difference.
