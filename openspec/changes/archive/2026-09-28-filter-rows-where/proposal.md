# Proposal: filter-rows-where

## Why

`Filter` and `FilterRows` call their function once per cell and keep a row when any one cell passes (#226, T-13 in `api-review.md`). The most common row condition, one column compared with another (`price < cost`), cannot be written with either: no single call sees both cells. The only way today is a CCL expression. `FilterByCustomElement` is `Filter` with the row and column arguments dropped, a second name for the same operation.

The documentation made it worse. `Docs/DataTable.md` called `Filter`'s second argument the column name and showed `columnIndex == "age"`, which never matches, because the argument is the Excel-style letter. Two `FilterRows` examples declared `func(colIndex, colName, x any)`, which does not compile against the method, and asserted `x.(int)`, which panics on any other cell.

## What Changes

- Add `DataTable.FilterRowsWhere(keep func(row *DataList) bool) *DataTable`. `keep` runs once per row with the row's cells in column order, named with the row's name; the result keeps the table's name, the column names and the kept rows' names. The row is a copy, a slice cell stays one cell, and a nil `keep` records an error and returns an empty table. `IDataTable` lists it.
- `FilterByCustomElement` is Deprecated in favour of `Filter` under the one-name rule, and removed in the next release. `TestFilterByCustomElementEqualsFilter` shows the two return the same table on twelve combinations of table and predicate; it passed on the code before this change.
- `Filter` and `FilterRows` work from the columns as they were when the call began. Their function runs inside the table's lock and may call the table's own methods; one that added a column crashed both with an index out of range. `FilterRowsWhere` does the same from the start.
- The doc comments and `Docs/DataTable.md` say plainly that `Filter` and `FilterRows` are any-cell tests and point to `FilterRowsWhere`; the wrong `Filter` and `FilterRows` examples are corrected.

The name follows the Filter family, where `FilterRows…` methods keep rows, and reads as the SQL clause it resembles: the rows where a condition holds.

## Capabilities

### New Capabilities

- `datatable-filtering`: what `Filter`, `FilterRows` and `FilterRowsWhere` keep, and the deprecated `FilterByCustomElement`.

### Modified Capabilities

(none)

## Impact

- `datatable_filters.go`, `interfaces.go`; new `datatable_filter_rows_where_test.go`.
- `Docs/DataTable.md`, both changelogs, `api-review.md`, `AGENTS.md` (removal follow-up), `delivery-status.md`.
- The agent skills name no filter method and are unchanged.
