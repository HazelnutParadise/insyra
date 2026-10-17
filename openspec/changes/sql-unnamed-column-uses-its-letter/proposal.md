# Proposal: sql-unnamed-column-uses-its-letter

## Why

`ToSQL` wrote a column with no name under the empty identifier `""`. SQLite accepts that name, but MySQL and PostgreSQL do not. This was measured on 2026-10-17 against MySQL 8.4 and PostgreSQL 16 running in Docker. The CLI script `newdl 1 2 3 as x`, `newdt x as t`, `save t sql m t2 if-exists replace` fails on MySQL with `Error 1166 (42000): Incorrect column name ''` and on PostgreSQL with `zero-length delimited identifier` (SQLSTATE 42601). `NewDataTable(NewDataList(...))` leaves every column unnamed, so the plainest table could not be saved to either server. The gap was recorded as an `AGENTS.md` follow-up on 2026-10-17.

insyra already refers to an unnamed column by its Excel-style letter. `Unpivot` labels such a column with its letter (`pivotColLabelForIndex`), and a fitted scaler remembers an unnamed column by its letter. pandas likewise writes every column under its label: an integer label `0` becomes the SQL column `"0"`.

## What Changes

- **BREAKING**: `ToSQL` and `ToSQLContext` write an unnamed column under its letter: `A`, `B`, and so on, counted among the table's columns, not including the row-name column. A table written to SQLite now gets a column `A` where it used to get `""`. Reading it back gives a column named `A`.
- When another column already has that letter as its name (compared without case, as MySQL and SQLite compare column names), the write is refused before anything is created. The error names the column.

## Capabilities

### Modified Capabilities

- `sql-database-connection`: how an unnamed column is named in the database.

## Impact

`datatable_to_sql.go`, its tests, the gated server tests, `Docs/DataTable.md`, both CHANGELOGs, and the `AGENTS.md` follow-up, which is removed.
