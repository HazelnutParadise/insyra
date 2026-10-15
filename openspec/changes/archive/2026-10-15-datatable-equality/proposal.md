# Proposal: datatable-equality

## Why

While ruling on [#220](https://github.com/HazelnutParadise/insyra/issues/220), the owner asked whether "the same" for a table includes every column and row name. `DataTable` has no comparison at all: only `DataList` has `IsEqualTo` and `IsTheSameAs`. To compare two tables today a caller has to walk the columns, call `DataList.IsEqualTo` on each, and compare the table, column and row names by hand.

The owner ruled on 2026-10-15 that tables get the same pair as lists, with `IsEqualTo` ignoring every name.

## What Changes

- New `DataTable.IsEqualTo(other *DataTable) bool`: true when both tables have the same number of rows and columns and every cell is equal under `DataList.IsEqualTo`'s rule (types must match, two NaN cells are equal). A cell past the end of a short column counts as `nil`. Names are ignored.
- New `DataTable.IsTheSameAs(other *DataTable) bool`: `IsEqualTo`, plus the same table name, the same column names in order, and the same row names in order. This is close to pandas' `DataFrame.equals`, which compares the row and column labels.
- Both return true for the same table, and false with a warning for a nil argument. Both lock the two tables together.
- `IDataTable` gains both methods.
- `Docs/DataTable.md`, both changelogs, `delivery-status.md`.

## Capabilities

### New Capabilities

- `datatable-equality`: what makes two tables equal and what makes them the same.

### Modified Capabilities

None.

## Impact

- `datatable_equality.go` (new), `interfaces.go`, a new test file. Nothing existing changes behaviour.
