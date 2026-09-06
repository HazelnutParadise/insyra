# Proposal: fix-api-review-batch-4

## Why

The second-round repository review (`api-review.md`, 2026-09-06) left 30 `severity:high` issues. Twenty-one of them have exactly one right fix and no API decision attached: a crafted workbook can truncate files outside the output directory, `ToCSV` reports success on a failed write, `AtomicDoAll` deadlocks when nested, eight CCL defects give wrong results or panic from a user expression, five CLI defects crash the session or leak database passwords, and five tests either cannot fail or never run. This change closes those twenty-one (#275, #276, #283, #284, #290, #299, #300, #301, #311, #312, #313, #314, #330, #341 range/keyword part, #342–#348). The nine that need an owner decision stay open.

## What Changes

- **CCL**: `NULL`/`TRUE`/`FALSE` are case-insensitive keywords; an Excel-style reference past the last column is an error; `@` yields a copy per row; a duration converts to seconds in numeric context; `MapContext` orders columns by name; nested sequence functions keep the whole column; absurd shifts, lengths and repeat counts are errors, and aggregate/sequence calls recover panics; the function registry is mutex-guarded; `SUM`/`AVG` skip `NaN`. `MaxResolvedColIndex` is added to `internal/ccl`. The name-before-index resolution (#341 remainder) and the variance-family "too few values" policy (CCL-28) are deferred: docs and an existing test pin the current contract.
- **Core**: `ToCSV` checks the final flush; `ToCSV`/`ToJSON` write through a temp file and rename; `AtomicDoN` runs inline (trust-zone) when the goroutine already holds one of the actors.
- **csvxl**: sheet names are validated as single path elements; rows are read before the output file exists; output goes through a temp file.
- **CLI**: `col`/`row`/`movavg`/`expsmooth`/`diff` never store nil; `NaN`/±Inf survive `SaveState`; root flags before raw-arg commands apply; `run` disables `OpenREPL` and bounds nesting; `db connect` lines are masked in every history path; history files are 0600.
- **Tests/CI/repo**: eleven `// TODO` DataList tests get assertions; two factor-analysis edge tests assert; `reference-verification.yml` runs every scikit-learn comparison; `insyra.test` is removed and `*.test` ignored; `delivery-status.md` is rewritten.
- Docs (`CCL.md`, `cli-dsl.md`, `DataTable.md`, `csvxl.md`, `DataList.md`, `AGENTS.md`), both changelogs, and `api-review.md` in the same change.

## Capabilities

### New Capabilities

- `ccl-evaluation-safety`: a user expression cannot panic, alias rows, read past the table, or silently mis-compare durations.
- `core-atomic-file-output`: `ToCSV`/`ToJSON` never leave a truncated file and never hide a write error.
- `core-multilock-reentry`: `AtomicDoAll` nested inside `AtomicDo` cannot deadlock.
- `csvxl-sheet-name-safety`: workbook sheet names cannot escape the output directory.
- `cli-session-robustness`: a failed command never crashes the session or corrupts the saved state.
- `cli-secret-hygiene`: database passwords never reach history or exports in clear text.
- `test-suite-integrity`: a test that exists either asserts or is removed, and every reference comparison has a workflow that runs it.

### Modified Capabilities

(none)

## Impact

- `internal/ccl/*`, `ccl.go`, `datatable_ccl.go`, `datatable_csv.go`, `datatable_json.go`, `write_atomic.go`, `internal/core/atomic.go`, `csvxl/convert.go`, `csvxl/convertDir.go`, `cli/root.go`, `cli/commands/{col,row,timeseries,run,registry,db_conn}.go`, `cli/repl/{repl,api}.go`, `cli/env/{state,manager}.go`, `.github/workflows/reference-verification.yml`, `.gitignore`, tests, docs, changelogs.
- No exported signature changes. `ExecContext` gains an unexported field. State files gain a new DataTable layout; the old string layout is still read.
