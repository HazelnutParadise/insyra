# Proposal: fix-api-review-batch-2

## Why

The second pass of `api-review.md` (DataTable, stats, mkt, cli, lp, datafetch) verified a further set of behaviours that are wrong rather than debatable: three `DataTable` methods panic on a bad index, `Data()` hands out the table's own backing slices, filtered tables share columns with the table they came from, `DropRowsByIndex(-1, 0)` deletes the wrong row, `Transpose` loses row names, the t/z/F tests count a blank cell in `n` while excluding it from the mean (turning p = 0.074 into p = 0.028), and `mkt.RFM` crashes the process on one non-numeric amount. None of these need an API decision; each has one correct behaviour its own documentation already implies. Fixing them now leaves the open design questions (LogFatal, the interfaces, failure shapes, name-vs-index) to be decided on a base that does not silently corrupt data.

## What Changes

- `DataTable.GetElementByNumberIndex`, `SetRowToColNames`, `SetColToRowNames` return nil / set `Err()` on a bad index instead of panicking. `Data()` and `ToMap()` return copies. Every `Filter*` method returns a table whose columns and row names are its own; a not-found result is `NewDataTable()`, not a bare struct; `FilterRows` and `FilterCols` use the table's row count and tolerate ragged columns.
- `DropRowsByIndex` normalises negative indices and de-duplicates before deleting, so `(-1, 0)` drops the first and last row and `(1, 1)` drops one row. `Transpose` carries every row name into the new column names. `ChangeRowName` to an existing name goes through `safeRowName` like every other row-name setter. `AppendRowsByColIndex` grows the table up to the requested column so the value is never dropped.
- `DropColsContainNumber` and `DropRowsContainNumber` recognise every Go numeric type. `DataTable.Mean` divides by the count of numeric cells (signature unchanged).
- `ToCSV` writes `time.Time` cells as RFC 3339 so `ParseDates` reads them back.
- The four CCL methods return the receiver, not nil, after a recovered panic.
- `SingleSampleTTest`, `TwoSampleTTest`, `SingleSampleZTest`, `TwoSampleZTest`, `FTestForVarianceEquality`, `BartlettTest`, `LeveneTest` read input through `numericSlice` and refuse an unreadable cell with the row named; `CalculateMoment` likewise.
- `mkt.RFM` treats a non-numeric amount as an unreadable row (warning, row skipped) instead of panicking; `RFM` and `CustomerActivityIndex` emit rows sorted by customer ID.
- `cli/env` rejects environment names containing path separators, `..`, or characters outside `[A-Za-z0-9._-]`.
- `lp`'s additional-info table and `datafetch`'s file geocode cache become deterministic and atomic respectively: fixed row order; write-to-temp-then-rename.
- **BREAKING (behavioural)**: inputs the hypothesis tests used to accept with a blank now return an error; `RFM`/`CAI` row order changes to sorted; filtered tables no longer mutate their source. Marked in both changelogs.
- Docs, both changelogs, and `api-review.md` (rows marked fixed) are updated in the same change.

## Capabilities

### New Capabilities

- `datatable-bounds-and-aliasing`: bad indices set `Err()`; `Data`/`ToMap`/`Filter*` never alias the table's storage.
- `datatable-row-operations`: `DropRowsByIndex`, `Transpose`, `ChangeRowName`, `AppendRowsByColIndex` behave as documented.
- `datatable-numeric-membership`: numeric membership uses one definition across `DropColsContainNumber`, `DropRowsContainNumber`, and `Mean`.
- `datatable-csv-time-roundtrip`: `ToCSV` and `ParseDates` round-trip `time.Time`.
- `datatable-ccl-failure-shape`: CCL methods never return nil.
- `stats-test-input`: t/z/F/Bartlett/Levene tests and `CalculateMoment` refuse unreadable cells.
- `mkt-input-and-order`: `RFM` never panics on input; `RFM`/`CAI` output order is deterministic.
- `cli-env-name`: environment names cannot escape the envs directory.
- `deterministic-and-atomic-output`: `lp` info table order is fixed; the file geocode cache is written atomically.

### Modified Capabilities

(none)

## Impact

- `datatable.go`, `datatable_filters.go`, `datatable_rowname.go`, `datatable_csv.go`, `datatable_ccl.go`, `stats/ttest.go`, `stats/ztest.go`, `stats/ftest.go`, `stats/moments.go`, `mkt/rfm.go`, `mkt/cai.go`, `cli/env/manager.go`, `lp/lp.go`, `datafetch/geocoding.go`, plus tests.
- `Docs/DataTable.md`, `Docs/stats.md`, `Docs/mkt.md`, `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`.
- No new dependencies. Signatures unchanged.
