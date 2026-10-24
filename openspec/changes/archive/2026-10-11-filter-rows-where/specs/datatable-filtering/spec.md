## ADDED Requirements

### Requirement: Filter and FilterRows keep a row when any cell passes

`Filter` and `FilterRows` SHALL call their function once per cell and SHALL keep a row as soon as one of its cells passes. `Filter`'s column argument SHALL be the column's Excel-style letter. `Filter` and `FilterRows` SHALL work from a copy of the columns as they were when the call began, so a predicate that modifies the table columns SHALL NOT crash or shift reading. Their documentation SHALL say that they cannot compare cells of the same row and SHALL point to `FilterRowsWhere`.

#### Scenario: The function modifies columns

- **WHEN** the function passed to `Filter`, `FilterRows` or `FilterRowsWhere` drops or appends a column in the table being filtered
- **THEN** the call reads the columns the table had when the call began and does not panic or shift columns

#### Scenario: Two cells of one row

- **WHEN** `Filter` is given a function that passes the value 3 or 4, on a table whose second row is `(5, 4)` and third row is `(3, 3)`
- **THEN** both rows are kept, each because one cell passed

### Requirement: FilterRowsWhere judges a whole row

`FilterRowsWhere(keep)` SHALL call `keep` once per row with a DataList holding that row's cells in column order and named with the row's name, and SHALL return a new table holding the rows for which `keep` returned true, with the source table's name, column names and the kept rows' names.

#### Scenario: One column compared with another

- **WHEN** a caller keeps the rows whose first cell is greater than the second, on rows `(1, 2)`, `(5, 4)`, `(3, 3)`, `(8, 1)`
- **THEN** the result holds `(5, 4)` and `(8, 1)` with their row names

#### Scenario: No row passes

- **WHEN** `keep` returns false for every row
- **THEN** the result has the source's columns and no rows

#### Scenario: Row DataList isolation

- **WHEN** the `keep` callback modifies cells of the passed DataList or receives a slice cell
- **THEN** modifying the list does not affect the source or result table, and slice cells remain intact

#### Scenario: Nil keep predicate

- **WHEN** `FilterRowsWhere(nil)` is called
- **THEN** an error is recorded on the table and an empty, non-nil DataTable is returned

### Requirement: FilterByCustomElement is a deprecated spelling of Filter

`FilterByCustomElement(f)` SHALL return the same table as `Filter` given a function that passes `f` the cell's value, and SHALL be marked Deprecated (removed in 0.4) with that replacement named.

#### Scenario: Same result

- **WHEN** both are applied with the same predicate to the same table
- **THEN** the data, column names, row names and table name are equal
