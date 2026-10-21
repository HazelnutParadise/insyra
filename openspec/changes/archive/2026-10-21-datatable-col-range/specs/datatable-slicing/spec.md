## ADDED Requirements

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
