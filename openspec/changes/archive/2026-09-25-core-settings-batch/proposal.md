## Why

The second batch of #213 applies the owner's 2026-09-25 rulings on how a function takes settings to the DataList and DataTable surface.

- A table-level fill skipped a column it could not fill even when the caller had named it (E-7), so `dt.FillWithMean(Name("city"))` on a text column looked like it worked. The owner ruled that a named column that cannot be filled is an error, while filling every column still skips what it cannot fill.
- `DataTable.FillByInterpolation` could not extrapolate at all, so the CLI's `extrapolate` option was dropped for tables.
- `NewSimpleImputer(strategy, constant ...any)` let a constant go with the mean or be forgotten for the constant strategy, and said so only at `Fit` (E-6). Its two settings are statistical, so they go in an options struct; the name stays scikit-learn's.
- `ShowRange` and its siblings ignored a third value or a value that is not an int and showed everything. The owner kept the compact form but made it strict, and added `ShowHead`/`ShowTail` as plainer spellings.
- `Sample(n, withReplacement, ...)` keeps `withReplacement` required: the owner ruled that a setting with no safe default stays a parameter (recorded in AGENTS.md).

## What Changes

- Table `FillWithMean`, `FillWithMedian`, `FillByInterpolation` and `FillWithMode` record an error naming the column and its data type for a named column they cannot fill, and still fill the other named columns.
- **BREAKING**: `DataTable.FillByInterpolation(extrapolate bool, cols ...any)`.
- **BREAKING**: `NewSimpleImputer(opts ...SimpleImputerOptions)` with `Strategy` (empty means mean) and `FillValue`.
- `ShowRange`, `ShowTypesRange` (DataList and DataTable) print an error line for arguments they cannot read; `ShowHead`, `ShowTail`, `ShowHeadTo`, `ShowTailTo` on both types.
- CLI `fillna` passes `extrapolate` to tables and fails, saving nothing, when the fill recorded an error.
- Not changed: `SimpleImputer`'s pass-through of a selected non-numeric column under mean or median, which is a designed, reported behaviour (`ScalerParams.PassThrough`) and needs its own ruling.

## Capabilities

### New Capabilities
- `table-fill`: what a table-level fill does with a column it cannot fill.
- `show-range`: what the display functions do with a range they cannot read.

### Modified Capabilities
None.

## Impact

- `datatable_impute.go`, `datatable_simple_imputer.go`, `show.go`, `show_head_tail.go` (new), `interfaces.go`, `cli/commands/fillna.go`.
- `Docs/DataTable.md`, `Docs/DataList.md`, `Docs/cli-dsl.md`, both skills, both changelogs.
