# Proposal: datatable-col-range

## Why

[#408 用欄名取一段欄](https://github.com/HazelnutParadise/insyra/issues/408): `SliceCols` is end-exclusive like a Go slice, so to take `open` through `volume` with `volume` kept a caller must name the column after it, which a name does not have. pandas `loc`, dplyr `open:volume`, Excel `B:D` and CCL `A:C` all include both ends.

## What Changes

- New `DataTable.ColRange(from, to any)`, also on `IDataTable`: the columns from `from` through `to`, both included.
- Bounds are column selectors (letter, `Name`, int), resolved by the existing selector rule, so the three spellings of a column are one bound. `nil` opens an end. A negative int counts from the end and names that column.
- Every bound must be a real column: unlike `SliceCols`, an int equal to `NumCols()` is an error.
- Errors and the empty result follow `SliceCols`: recorded on `Err()`, empty non-nil table. Short columns are padded with `nil` as in `SliceCols`.
- `SliceCols` is unchanged. A row version and CLI/DSL exposure are out of scope.

## Capabilities

### Modified Capabilities

- `datatable-slicing`: adds the inclusive column range.

## Impact

`datatable_filters.go` (a private helper shared with `SliceCols`), `interfaces.go`, `datatable_colrange_test.go`, `Docs/DataTable.md`, both CHANGELOGs. Additive; no existing call changes.
