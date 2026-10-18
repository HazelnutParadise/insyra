## Why

`ToSQL` wrote a column with no name under the empty identifier `""`. SQLite accepts that name, but MySQL and PostgreSQL do not, so `NewDataTable(NewDataList(...))`, whose columns are all unnamed, could not be saved to either server. insyra already refers to an unnamed column by its Excel-style letter. This is `0.4`'s `sql-unnamed-column-uses-its-letter`, taken on this line by the owner's decision of 2026-10-18 so the three databases agree.

## What Changes

- **BREAKING**: `ToSQL` and `ToSQLContext` write an unnamed column under its letter (`A`, `B`, …), counted among the table's columns without the row-name column. A table written to SQLite gets `A` where it used to get `""`.
- When another column already has that letter as its name, compared without case, the write is refused before anything is created.

## Impact

`datatable_to_sql.go`, its tests, `Docs/DataTable.md`, both CHANGELOGs.
