# Proposal: datatable-col-range

## Why

[#408 用欄名取一段欄](https://github.com/HazelnutParadise/insyra/issues/408): `SliceCols` is end-exclusive like a Go slice, so a caller who wants columns `B` through `D` with `D` kept must name the column after `D`, which may not exist. pandas `loc`, Excel `B:D` and CCL `A:C` all include both ends.

## What Changes

- New `DataTable.ColRange(from, to any)`, also on `IDataTable`: the columns from `from` through `to`, both included.
- Bounds are an Excel-style letter or an int position, the same bound for the two spellings of a column. `nil` opens an end. A negative int counts from the end and names that column.
- Every bound must be a real column: unlike `SliceCols`, an int equal to `NumCols()` is an error.
- Errors and the empty result follow `SliceCols`: a warning recorded on the table and an empty, non-nil table. Short columns are padded with `nil`.
- `SliceCols` is unchanged. A row version and CLI/DSL exposure are out of scope.

## Capabilities

### Modified Capabilities

- `datatable-slicing`: adds the inclusive column range.

## Impact

`datatable_filters.go` (a private helper shared with `SliceCols`), `interfaces.go`, `datatable_colrange_test.go`, `Docs/DataTable.md`, both CHANGELOGs. Additive; no existing call changes.
