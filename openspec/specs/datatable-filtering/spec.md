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

`FilterRowsWhere(keep)` SHALL call `keep` once per row with a DataList holding that row's cells in column order and named with the row's name, and SHALL return a new table holding the rows for which `keep` returned true, with the source table's name, column names and the kept rows' names. A nil `keep` SHALL record an error on the table and return an empty, non-nil DataTable.

#### Scenario: One column compared with another

- **WHEN** a caller keeps the rows whose first cell is greater than the second, on rows `(1, 2)`, `(5, 4)`, `(3, 3)`, `(8, 1)`
- **THEN** the result holds `(5, 4)` and `(8, 1)` with their row names

#### Scenario: No row passes

- **WHEN** `keep` returns false for every row
- **THEN** the result has the source's columns and no rows

### Requirement: A predicate that drops a column does not shift the columns it reads

`Filter`, `FilterRows` and `FilterRowsWhere` SHALL build their result from the columns the table had when the call began, so dropping a column from the table inside the predicate or the keep function MUST NOT change the values the result holds.

#### Scenario: A column dropped during the call

- **WHEN** the predicate of `Filter`, `FilterRows` or `FilterRowsWhere` drops column `A` the first time it runs on the table with rows `(1, 2)`, `(5, 4)`, `(3, 3)`, `(8, 1)` and keeps every row
- **THEN** the result holds all four rows with both original columns

### Requirement: The DataList handed to keep is a copy

The DataList that `FilterRowsWhere` passes to `keep` SHALL be a copy of the row: setting one of its cells changes neither the source table nor the result.

#### Scenario: keep sets a cell of its row

- **WHEN** `keep` sets one cell of the row it is given to a new value
- **THEN** the source table and the result keep the original value

### Requirement: A slice or map cell in a row is shared with the source

A cell of the row that `FilterRowsWhere` passes to `keep` and that holds a slice or map SHALL share that value with the source table, so changing the contents of that value MUST change the source table and the result too.

#### Scenario: keep writes into a slice cell of its row

- **WHEN** `keep` writes into the slice held by one of its row's cells
- **THEN** the source table's cell shows the write

### Requirement: A slice cell stays one cell in the row handed to keep

A slice cell of the table SHALL stay one cell in the row that `FilterRowsWhere` passes to `keep`, and MUST NOT be split into one cell per element.

#### Scenario: A slice cell in a row

- **WHEN** a table holds a `[]byte` cell in a row and `keep` is called for that row
- **THEN** the row handed to `keep` holds that slice as one cell
