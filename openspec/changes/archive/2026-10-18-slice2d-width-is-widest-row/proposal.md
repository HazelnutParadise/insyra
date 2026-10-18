# Proposal: slice2d-width-is-widest-row

## Why

[#410](https://github.com/HazelnutParadise/insyra/issues/410): `ReadSlice2D` took the column count from the first row, so `[][]float64{{1}, {2, 3}}` became a 2x1 table and the `3` vanished with no error. The owner chose, on 2026-10-18, to make the table as wide as the widest row.

## What Changes

- `ReadSlice2D` / `Slice2DToDataTable` size the table by the longest row. Shorter rows are still padded with nil. An empty first row is still an error.
- `TestSlice2DToDataTable_InconsistentLengths`, which pinned the first-row width, now expects 4 columns.

## Capabilities

### New Capabilities

- `core-data-input`: how 2D slices become tables.

## Impact

`read.go`, `read_test.go`, `Docs/DataTable.md`, both CHANGELOGs. A result changes for ragged input only, and only by keeping values that were dropped.
