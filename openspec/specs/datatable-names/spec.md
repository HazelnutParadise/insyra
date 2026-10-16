# datatable-names Specification

## Purpose
How a DataTable names its columns and rows. The name setters only name what exists. A column or row is never treated as clashing with its own name. A real clash gets a numeric suffix and a warning, and never touches `Err()`.

## Requirements

### Requirement: SetColNames and SetRowNames name only what exists

`DataTable.SetColNames` and `DataTable.SetRowNames` SHALL name the existing columns or rows in order. They SHALL NOT add a column or row. Names past the last column or row SHALL be ignored, and a warning listing them SHALL be logged without touching `Err()`. Columns or rows past the last name SHALL lose their names.

#### Scenario: More column names than columns
- **WHEN** `SetColNames([]string{"x", "y", "z"})` is called on a table with two columns
- **THEN** the table still has two columns, named `x` and `y`, a warning lists `z`, and `Err()` is nil

#### Scenario: More row names than rows
- **WHEN** `SetRowNames([]string{"r1", "r2", "r3"})` is called on a two-row table
- **THEN** the rows are named `r1` and `r2`, and a warning lists `r3`

#### Scenario: Fewer names than columns
- **WHEN** `SetColNames([]string{"x"})` is called on a table whose columns A and B are named `price` and `qty`
- **THEN** the names are `x` and `""`

### Requirement: A column or row's own name is never a clash

A setter that renames a column or row SHALL NOT count that column's or row's current name as taken. `SetColNames` and `SetRowNames`, which rename every column or row, SHALL check each name only against names given earlier in the same call.

#### Scenario: The same names again
- **WHEN** `SetColNames([]string{"price", "qty"})` is called on a table whose columns A and B are named `price` and `qty`
- **THEN** the names stay `price`, `qty`

#### Scenario: Two columns trade names
- **WHEN** `SetColNames([]string{"qty", "price"})` is called on a table whose columns A and B are named `price` and `qty`, or a row holding `qty`, `price` is passed to `SetRowToColNames` on that table
- **THEN** column A is named `qty` and column B `price`, and no cell moves

#### Scenario: A single setter given the column's own name
- **WHEN** `SetColNameByIndex("A", "price")`, `SetColNameByNumber(1, "qty")` or `ChangeColName("price", "price")` is called on a table whose columns A and B are named `price` and `qty`, or `SetRowNameByIndex(0, name)` is called with the row's own name
- **THEN** no name changes

### Requirement: A clashing name gets a suffix and a warning

When a column or row name another column or row already has is assigned, the name SHALL get the first free numeric suffix (`_1`, `_2`, …). A warning naming the original and the new name SHALL be logged, and `Err()` SHALL NOT be touched.

#### Scenario: Appending a column with a taken name
- **WHEN** `AppendCols` adds a column named `price` to a table whose columns are named `price` and `qty`
- **THEN** the names are `price`, `qty`, `price_1`, a warning names `price_1`, and `Err()` is nil

#### Scenario: A name repeated in one call
- **WHEN** `SetColNames([]string{"x", "x"})` or `SetRowNames([]string{"r", "r"})` is called
- **THEN** the second name becomes `x_1` or `r_1`, with a warning

### Requirement: The CLI refuses a name count that does not match

The `setcolnames` and `setrownames` commands SHALL fail, leaving the table unchanged, when the number of names differs from the number of columns or rows.

#### Scenario: Too many row names
- **WHEN** `setrownames dt r1 r2 r3 r4` runs on a three-row table
- **THEN** the command returns an error
