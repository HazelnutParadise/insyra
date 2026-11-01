## Why

`WEEKDAY`, `TRIM` and `COUNT` share a name with Excel functions and answer differently, which CCL's Excel look-alike syntax makes easy to miss. The owner ruled on 2026-11-01 (#414) that they follow Excel.

## What Changes

- **BREAKING** `WEEKDAY` counts Sunday as 1 through Saturday as 7.
- **BREAKING** `TRIM` also turns each run of spaces inside the text into one.
- **BREAKING** `COUNT` counts numbers and dates only; the new `COUNTA` counts every value that is not nil, including in the Parquet streaming path.

## Impact

- `internal/ccl/`, `Docs/CCL.md`, `Docs/DataTable.md`, `Docs/parquet.md`, both changelogs, `ccl-type-semantics` spec.
