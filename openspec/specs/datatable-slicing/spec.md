# datatable-slicing Specification

## Purpose
How `SliceRows` and `SliceCols` take a contiguous range of rows or columns, which bounds they accept, and how the index filters they replace relate to them. `ColRange` takes a range of columns with both ends included, like Excel's `B:D`.

## Requirements

### Requirement: SliceRows takes a row range by position

`SliceRows(from, to)` SHALL return rows `from` through `to-1` as a new DataTable, for `0 <= from <= to <= NumRows()`.

#### Scenario: Two middle rows

- **WHEN** `SliceRows(1, 3)` is called on a five-row table whose rows 1 and 2 are named `one` and `two`
- **THEN** the result holds those two rows, named `one` and `two`, with every column and the table's name

#### Scenario: Empty row slice

- **WHEN** `SliceRows(2, 2)` is called on a five-row table
- **THEN** the result has no rows, keeps every column, and retains the table's name

### Requirement: SliceCols takes a column range by position

`SliceCols(from, to)` SHALL return columns `from` up to, but not including, `to` as a new DataTable, keeping every row and padding short columns with `nil`.

#### Scenario: Two middle columns

- **WHEN** `SliceCols(1, 3)` and `SliceCols("B", "D")` are called on a four-column table
- **THEN** each result holds columns B and C with every row and the row names

#### Scenario: Open ends

- **WHEN** `SliceCols("C", nil)` and `SliceCols(nil, "C")` are called on the same table
- **THEN** the first holds C and D and the second A and B

#### Scenario: Ragged table padded with nil

- **WHEN** `SliceCols` is called on a table whose columns differ in length
- **THEN** the result has `NumRows()` rows and shorter columns are padded with `nil`

#### Scenario: Empty column slice

- **WHEN** `SliceCols(1, 1)` or `SliceCols("B", "B")` is called
- **THEN** the result has no columns

### Requirement: Slicing preserves metadata and owns its data

A sliced DataTable SHALL keep the source table's name, the column names and the names of the rows it holds, and SHALL own its data.

#### Scenario: Sliced table independence

- **WHEN** data in a sliced table is modified after `SliceRows` or `SliceCols`
- **THEN** the source table's data remains unmodified

### Requirement: A bound outside the table is an error

For `SliceRows`, a bound with `from < 0`, `from > to` or `to` past the length SHALL record an error on the table naming the bounds. For `SliceCols`, an int outside `-NumCols()` to `NumCols()`, an unparseable letter, a value of another type, or a `from` that comes after `to` SHALL record an error on the table. The method SHALL then return an empty, non-nil DataTable and SHALL NOT panic.

#### Scenario: Past the last row

- **WHEN** `SliceRows(0, 6)` is called on a five-row table
- **THEN** `Err()` reports that the bounds are out of range, and the result has no rows and no columns

#### Scenario: The ends reversed

- **WHEN** `SliceCols("C", "B")` is called
- **THEN** `Err()` says that from comes after to

### Requirement: The index filters are deprecated spellings of slice methods

The ten `FilterColsByColIndex…` and `FilterRowsByRowIndex…` methods SHALL be marked Deprecated (removed in 0.4), naming their slice replacement and returning the same data and names for every bound inside the table.

#### Scenario: Every threshold inside the table

- **WHEN** each deprecated index method is called with each column letter or row index of a four-column, five-row table
- **THEN** its result equals the corresponding slice

#### Scenario: A letter past the last column

- **WHEN** `FilterColsByColIndexLessThan("Z")` is called on a four-column table
- **THEN** it returns every column and does not panic

### Requirement: Headers and SetHeaders are deprecated spellings of column naming methods

`Headers` and `SetHeaders` SHALL be marked Deprecated in favour of `ColNames` and `SetColNames` (removed in 0.4) and behave identically to them.

#### Scenario: Deprecated header methods match column naming methods

- **WHEN** `Headers()` and `SetHeaders(...)` are called
- **THEN** their behavior and return values match `ColNames()` and `SetColNames(...)`

### Requirement: ColRange takes columns from one to another, both included

`ColRange(from, to)` SHALL return the columns from `from` through `to`, both included, as a new DataTable that keeps every row, the row names and the table's name and owns its data. `from` and `to` naming the same column SHALL give that one column.

#### Scenario: Two spellings of a range

- **WHEN** `ColRange("B", "D")` and `ColRange(1, 3)` are called on a four-column table whose columns are named `a` to `d`
- **THEN** each result holds columns B, C and D

#### Scenario: One column

- **WHEN** `ColRange("B", "B")` is called
- **THEN** the result holds column B, where `SliceCols("B", "B")` holds none

### Requirement: A ColRange bound is a column the table has

Each bound of `ColRange` SHALL be an Excel-style letter or an int position, and the two spellings of one column SHALL be the same bound. A negative int SHALL count from the end and name that column. `nil` SHALL be the first column as `from` and the last as `to`. An int equal to `NumCols()` SHALL be out of range.

#### Scenario: Open ends and negative ints

- **WHEN** `ColRange("C", nil)`, `ColRange(nil, "B")`, `ColRange(nil, nil)` and `ColRange(-2, nil)` are called on the same table
- **THEN** the results hold C and D, A and B, all four columns, and C and D

#### Scenario: One past the last column

- **WHEN** `ColRange(0, 4)` is called on the same table
- **THEN** `Err()` says the bound is out of range and the result is empty

### Requirement: A ColRange bound outside the table is an error

A bound that picks no column, an int outside `-NumCols()` to `NumCols()-1`, a value of another type, a `from` that comes after `to`, or a table with no columns SHALL record an error on the table. `ColRange` SHALL then return an empty, non-nil DataTable and SHALL NOT panic.

#### Scenario: A letter past the last column

- **WHEN** `ColRange("A", "Z")` is called on the same table
- **THEN** `Err()` says column Z does not exist, and the result is empty

#### Scenario: The ends reversed

- **WHEN** `ColRange("C", "B")` is called
- **THEN** `Err()` says that from comes after to

### Requirement: ColRange pads a short column with nil

When a column that `ColRange` selects is shorter than the table's longest column, the result SHALL pad it with `nil` to that length, and the source table SHALL NOT change.

#### Scenario: A short column in the range

- **WHEN** `ColRange(0, 1)` is called on a table whose column `A` holds three values and whose column `B` holds one value
- **THEN** the result holds both columns with three rows, the second `"x"`, `nil`, `nil`, and the source is unchanged
