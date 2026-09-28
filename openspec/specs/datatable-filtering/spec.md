# datatable-filtering Specification

## Purpose
Says which rows the `DataTable` filters keep. `Filter` and `FilterRows` judge one cell at a time and keep a row when any cell passes; `FilterRowsWhere` judges a whole row, so a condition can compare cells of the same row.
## Requirements
### Requirement: Filter and FilterRows keep a row when any cell passes

`Filter` and `FilterRows` SHALL call their function once per cell and SHALL keep a row as soon as one of its cells passes. `Filter`'s column argument SHALL be the column's Excel-style letter. Their documentation SHALL say that they cannot compare cells of the same row and SHALL point to `FilterRowsWhere`.

#### Scenario: The function adds a column

- **WHEN** the function passed to `Filter`, `FilterRows` or `FilterRowsWhere` appends a column to the table being filtered
- **THEN** the call returns the rows with the columns the table had when the call began, and does not panic

#### Scenario: Two cells of one row

- **WHEN** `Filter` is given a function that passes the value 3 or 4, on a table whose second row is `(5, 4)` and third row is `(3, 3)`
- **THEN** both rows are kept, each because one cell passed

### Requirement: FilterRowsWhere judges a whole row

`FilterRowsWhere(keep)` SHALL call `keep` once per row with a DataList holding that row's cells in column order and named with the row's name, and SHALL return a new table holding the rows for which `keep` returned true, with the source table's name, column names and the kept rows' names. The DataList SHALL be a copy, so changing it SHALL NOT change the source or the result, and a slice cell SHALL stay one cell. A nil `keep` SHALL record an error on the table and return an empty, non-nil DataTable.

#### Scenario: One column compared with another

- **WHEN** a caller keeps the rows whose first cell is greater than the second, on rows `(1, 2)`, `(5, 4)`, `(3, 3)`, `(8, 1)`
- **THEN** the result holds `(5, 4)` and `(8, 1)` with their row names

#### Scenario: No row passes

- **WHEN** `keep` returns false for every row
- **THEN** the result has the source's columns and no rows

### Requirement: FilterByCustomElement is a deprecated spelling of Filter

`FilterByCustomElement(f)` SHALL return the same table as `Filter` given a function that passes `f` the cell's value, and SHALL be marked Deprecated with that replacement named.

#### Scenario: Same result

- **WHEN** both are applied with the same predicate to the same table
- **THEN** the data, column names, row names and table name are equal

