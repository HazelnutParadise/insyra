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

## Backport to dev (0.3.x)

Dev received:
- CCL keywords in any case, `@` row copies, durations as seconds, nested sequence functions, bounded arguments, the locked function registry, and name-ordered `NewMapContext` (`ccl-evaluation-safety`, trimmed).
- `ToCSV` returning its final flush error (`core-atomic-file-output`, trimmed to that requirement).
- `csvxl` refusing unsafe sheet names and reading a sheet before creating its CSV (`csvxl-sheet-name-safety`, trimmed).
- The CLI nil-result checks, NaN-safe state, root flags before raw-arg commands, script-safe `env open` with the nesting limit (`cli-session-robustness`, adapted), and masked `db connect` history at 0600 (`cli-secret-hygiene`).
- The DataList TODO test assertions that hold on dev, the factor-analysis edge assertions, the scikit-learn workflow pattern, and the `insyra.test` removal (`test-suite-integrity`).

Adapted on dev:
- The shift, window and `REPEAT` guards refuse only what differs by platform or panics (backport review): `NaN`, infinities, values outside int64, a negative repeat count and a `REPEAT` result whose length overflows `int`. The int32 bound and the 64 MiB `REPEAT` cap refused arguments v0.3.2 handled: `LAG(A, 3000000000)` and `ROLLING_MEAN(A, 3000000000)` gave nil cells, `LEN(REPEAT('', 100000000))` gave 0 and `REPEAT('ab', 40000000)` an 80 MB string. A shift is clamped to the column length and the rolling buffer to the row count, so those values neither overflow nor over-allocate.
- `ExcelToCsv`/`EachExcelToCsv` check the CSV file name actually used, per sheet before that sheet's file is written, and refuse only a name holding a path separator or leaving the output directory (backport review). Refusing the sheet names `.` and `..` rejected workbooks v0.3.2 converted to `..csv` and `...csv`, while a `csvNames` entry was never checked. The delta spec's temp-file clause, which describes 0.4, is replaced by dev's read-before-create rule.
- `ToCSV` keeps creating the file directly and returns the flush error from `writeCSV`, and, when the write succeeded, the error from closing the file (backport review: it was discarded by a deferred `Close`); `saveSheetAsCsv` reads the sheet first, creates the CSV directly and returns its flush error.
- A DataTable variable uses the column layout only when a cell is NaN or ±Inf; every other variable, including finite `float32` cells, is written byte-for-byte as before.
- `datalist_test.go` drops the nil-cell `Normalize` and one-value `Standardize` assertions, which describe batch 1's rework.

Stayed on 0.4:
- The recovers in `callAggregateFunction` and `callSequenceFunction` (backport review): a registered aggregate or sequence function that panics would make `AddColUsingCCL` and the `EditCol*UsingCCL` methods return the receiver instead of nil, a changed return value. On dev the method's own recover handles it, as on v0.3.2; the built-in shift and `REPEAT` guards return errors without it.
- `AtomicDoN` running the callback inline when nested in `AtomicDo`: the other instances went unlocked, a data race v0.3.2 did not have. Dev keeps skipping the held actors and locking the rest; `core-multilock-reentry` and its delta spec here were rewritten to say so (backport review).
- `SUM`/`AVG` and `collectFloats` skipping NaN: changes returned values.
- An Excel-style reference past the last column becoming an error (`MaxResolvedColIndex`, `checkCCLColRange`): starts returning an error.
- `ToCSV`/`ToJSON` and `csvxl` CSVs written through a temp file and rename (`write_atomic.go`): changes how files are produced; owner decision.
- Every DataTable variable moving to the column layout in state.json: changes file output.
