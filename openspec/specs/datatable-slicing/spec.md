# datatable-slicing Specification

## Purpose
Says how `SliceRows` and `SliceCols` take a contiguous range of rows or columns by position, with Go slice bounds and an error for a bound outside the table, and which older names are deprecated spellings of them.
## Requirements
### Requirement: SliceRows and SliceCols take a range by position

`SliceRows(from, to)` SHALL return rows `from` through `to-1` and `SliceCols(from, to)` columns `from` through `to-1` as a new DataTable, for `0 <= from <= to <= NumRows()` or `NumCols()` respectively. The result SHALL keep the source table's name, the column names and the names of the rows it holds, and SHALL own its data. `from == to` SHALL give no rows (with every column) or no columns.

#### Scenario: Two middle rows

- **WHEN** `SliceRows(1, 3)` is called on a five-row table whose rows 1 and 2 are named `one` and `two`
- **THEN** the result holds those two rows, named `one` and `two`, with every column and the table's name

#### Scenario: Two middle columns

- **WHEN** `SliceCols(1, 3)` is called on a four-column table
- **THEN** the result holds columns B and C with every row and the row names

### Requirement: A bound outside the table is an error

A bound with `from < 0`, `from > to` or `to` past the length SHALL record an error on the table naming the bounds, and the method SHALL return an empty, non-nil DataTable. It SHALL NOT panic.

#### Scenario: Past the last row

- **WHEN** `SliceRows(0, 6)` is called on a five-row table
- **THEN** `Err()` reports that the bounds are out of range, and the result has no rows and no columns

### Requirement: The index filters and Headers are deprecated spellings

The ten `FilterColsByColIndex…` and `FilterRowsByRowIndex…` methods SHALL be marked Deprecated, each naming the `SliceCols` or `SliceRows` call it stands for, and SHALL return the same data, column names and row names as that call for every bound inside the table. `FilterColsByColIndexLessThan` and `FilterColsByColIndexLessThanOrEqualTo` SHALL keep every column for a letter past the last one instead of panicking. `Headers` and `SetHeaders` SHALL be marked Deprecated in favour of `ColNames` and `SetColNames` and behave exactly like them.

#### Scenario: Every threshold inside the table

- **WHEN** each deprecated index method is called with each column letter or row index of a four-column, five-row table
- **THEN** its result equals the corresponding slice

#### Scenario: A letter past the last column

- **WHEN** `FilterColsByColIndexLessThan("Z")` is called on a four-column table
- **THEN** it returns every column and does not panic

