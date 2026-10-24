# Proposal: datatable-equality

## Why

`DataTable` has no comparison: only `DataList` has `IsEqualTo` and `IsTheSameAs`. To compare two tables a caller has to walk the columns and compare the names by hand. The owner ruled on 2026-10-15 on `0.4` that tables get the same pair as lists, with `IsEqualTo` ignoring every name. The change only adds methods, so it is ported to the 0.3.x line.

## What Changes

- New `DataTable.IsEqualTo(other *DataTable) bool`: true when both tables have the same number of rows and columns and every cell is equal. Types must match, as in `DataList.IsEqualTo`, and two NaN cells are equal. A cell past the end of a short column counts as `nil`. Names are ignored.
- New `DataTable.IsTheSameAs(other *DataTable) bool`: `IsEqualTo`, plus the same table name, column names and row names, in order.
- Both return true for the same table, and false with a warning for a nil argument. Both lock the two tables together.
- `IDataTable` gains both methods.
- On this line `DataList.IsEqualTo` still treats two NaN cells as different; the table method treats them as equal, which is what `0.4` does for both.

## Capabilities

### New Capabilities

- `datatable-equality`: what makes two tables equal and what makes them the same.

### Modified Capabilities

None.

## Impact

- `datatable_equality.go` (new), `interfaces.go`, `Docs/DataTable.md`, both changelogs, and a new test file. Nothing existing changes behaviour.
