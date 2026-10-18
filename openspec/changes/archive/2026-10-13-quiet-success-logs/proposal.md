# Proposal: quiet-success-logs

## Why

api-review C-9 (#215), ported from `0.4` to the 0.3.x line. Several functions logged at Info that they had succeeded, which their nil error already says. The CLI's `convert` and `plot` printed their own confirmation right after the library's, so a script run with the default `--log-level info` showed both. The owner ruled on 2026-10-13 to keep `LogLevelInfo` as the default, delete those lines, and write the rule down so new code follows it. Fewer log lines is not a change to any return value.

## What Changes

- **Lines removed:** the Info lines of `csvxl`'s four conversion functions (the deprecated spellings included), `plot.SaveHTML`, `plot.SavePNG`, `gplot.SaveChart`, and `py.PipInstall`, `PipUninstall` and `PipList`. Each function returns what it returned before. The `csvxl` functions keep their encoding-detection line.
- **New Key Convention in `AGENTS.md`:** log at Info only what the return value does not say. A follow-up records that encoding detection is unstable on a very short Big5 file.
- **`Docs/Configuration.md`:** says what insyra logs at Info.

## Capabilities

### New Capabilities

- `log-content`: what insyra writes to its log at Info level.

### Modified Capabilities

None.

## Impact

- `csvxl/convert.go`, `csvxl/convertDir.go`, `plot/save_chart.go`, `gplot/save_chart.go`, `py/py.go`, `success_logs_test.go`, `AGENTS.md`, `Docs/Configuration.md` and both changelogs.
- **Who notices:** a program or script that read these lines in its log.
