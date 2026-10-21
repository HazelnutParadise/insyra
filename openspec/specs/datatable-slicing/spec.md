# datatable-slicing Specification

## Purpose
Says how `SliceRows` and `SliceCols` take a contiguous range of rows or columns with half-open Go slice bounds, `SliceRows` by position and `SliceCols` by column selector, what happens to a bound they cannot use, and which older names are deprecated spellings of them. `ColRange` takes a range of columns by selector with both ends included, like Excel's `B:D`.

## Requirements

### Requirement: SliceRows and SliceCols take a range by position

`SliceRows(from, to)` SHALL return rows `from` through `to-1` as a new DataTable, for `0 <= from <= to <= NumRows()`. `SliceCols(from, to)` SHALL return the columns from `from` up to, but not including, `to`. The result SHALL keep the source table's name, the column names and the names of the rows it holds, and SHALL own its data. `from == to` SHALL give no rows (with every column) or no columns.

#### Scenario: Two middle rows

- **WHEN** `SliceRows(1, 3)` is called on a five-row table whose rows 1 and 2 are named `one` and `two`
- **THEN** the result holds those two rows, named `one` and `two`, with every column and the table's name

### Requirement: A SliceCols bound is any column selector

Each bound of `SliceCols` SHALL be a column selector: an Excel-style letter, a `Name`, or an int position. The spellings of one column SHALL be the same bound, and `to` SHALL stay excluded whichever spelling is used. `nil` SHALL be the first column as `from` and one past the last as `to`, and an int bound SHALL count from the end when negative and MAY be `NumCols()`.

#### Scenario: Two middle columns

- **WHEN** `SliceCols(1, 3)`, `SliceCols("B", "D")` and `SliceCols(Name("b"), Name("d"))` are called on a four-column table whose columns are named `a` to `d`
- **THEN** each result holds columns B and C with every row and the row names

#### Scenario: Open ends

- **WHEN** `SliceCols("C", nil)` and `SliceCols(nil, "C")` are called on the same table
- **THEN** the first holds C and D and the second A and B

### Requirement: A bound outside the table is an error

For `SliceRows`, a bound with `from < 0`, `from > to` or `to` past the length SHALL record an error on the table naming the bounds. For `SliceCols`, a bound that picks no column, an int outside `-NumCols()` to `NumCols()`, a value of another type, or a `from` that comes after `to` SHALL record an error on the table. The method SHALL then return an empty, non-nil DataTable and SHALL NOT panic.

#### Scenario: Past the last row

- **WHEN** `SliceRows(0, 6)` is called on a five-row table
- **THEN** `Err()` reports that the bounds are out of range, and the result has no rows and no columns

#### Scenario: An unknown name

- **WHEN** `SliceCols(Name("nope"), nil)` is called
- **THEN** `Err()` says no column is named nope, and the result is empty

#### Scenario: The ends reversed

- **WHEN** `SliceCols("C", "B")` is called
- **THEN** `Err()` says that from comes after to

### Requirement: SliceCols pads a short column with nil

When a column that `SliceCols` selects is shorter than the table's longest column, the result SHALL pad that column with `nil` to the table's longest column length, so the result keeps every row of the table, and the source table SHALL NOT change.

#### Scenario: A short column in the range

- **WHEN** `SliceCols(1, 2)` is called on a table whose column `A` holds three values and whose column `B` holds one value, `"x"`
- **THEN** the result holds one column of three values, `"x"`, `nil` and `nil`, and the source table is unchanged

### Requirement: ColRange takes columns from one to another, both included
`ColRange(from, to)` SHALL return the columns from `from` through `to`, both included, as a new DataTable that keeps every row, the row names and the table's name and owns its data. `from` and `to` naming the same column SHALL give that one column.

#### Scenario: Three spellings of a range
- **WHEN** `ColRange("B", "D")`, `ColRange(1, 3)` and `ColRange(Name("b"), Name("d"))` are called on a four-column table whose columns are named `a` to `d`
- **THEN** each result holds columns B, C and D

#### Scenario: One column
- **WHEN** `ColRange("B", "B")` is called
- **THEN** the result holds column B, where `SliceCols("B", "B")` holds none

### Requirement: A ColRange bound is a column the table has
Each bound of `ColRange` SHALL be a column selector, and the spellings of one column SHALL be the same bound. An int bound SHALL count from the end when negative and SHALL name that column. `nil` SHALL be the first column as `from` and the last as `to`. An int equal to `NumCols()` SHALL be out of range.

#### Scenario: Open ends and negative ints
- **WHEN** `ColRange("C", nil)`, `ColRange(nil, "B")`, `ColRange(nil, nil)` and `ColRange(-2, nil)` are called on the same table
- **THEN** the results hold C and D, A and B, all four columns, and C and D

#### Scenario: One past the last column
- **WHEN** `ColRange(0, 4)` is called on the same table
- **THEN** `Err()` says the bound is out of range and the result is empty

### Requirement: A ColRange bound outside the table is an error
A bound that picks no column, an int outside `-NumCols()` to `NumCols()-1`, a value of another type, or a `from` that comes after `to` SHALL record an error on the table. `ColRange` SHALL then return an empty, non-nil DataTable and SHALL NOT panic.

#### Scenario: An unknown name
- **WHEN** `ColRange(Name("nope"), nil)` is called
- **THEN** `Err()` says no column is named nope, and the result is empty

#### Scenario: The ends reversed
- **WHEN** `ColRange("C", "B")` is called
- **THEN** `Err()` says that from comes after to

### Requirement: ColRange pads a short column with nil
When a column that `ColRange` selects is shorter than the table's longest column, the result SHALL pad it with `nil` to that length, and the source table SHALL NOT change.

#### Scenario: A short column in the range
- **WHEN** `ColRange(0, 1)` is called on a table whose column `A` holds three values and whose column `B` holds one value
- **THEN** the result holds both columns with three rows, the second `"x"`, `nil`, `nil`, and the source is unchanged
