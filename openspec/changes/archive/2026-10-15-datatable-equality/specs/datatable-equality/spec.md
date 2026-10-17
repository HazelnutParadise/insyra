## ADDED Requirements

### Requirement: IsEqualTo compares shape and cells

`DataTable.IsEqualTo(other)` SHALL return true exactly when both tables have the same number of rows and columns and every cell is equal, with types compared as `DataList.IsEqualTo` does and two NaN cells equal. A cell past the end of a short column SHALL count as `nil`. Table, column and row names SHALL NOT be compared.

#### Scenario: Same cells, different names
- **WHEN** two tables hold the same cells under different table, column and row names
- **THEN** `IsEqualTo` is true and `IsTheSameAs` is false

#### Scenario: A cell of another type
- **WHEN** two tables differ in one cell, `int(1)` against `int64(1)`
- **THEN** `IsEqualTo` is false

#### Scenario: NaN cells
- **WHEN** two tables hold `math.NaN()` in the same cell and are otherwise equal
- **THEN** `IsEqualTo` is true

#### Scenario: A different shape
- **WHEN** one table has an extra row or an extra column
- **THEN** `IsEqualTo` is false

### Requirement: IsTheSameAs adds every name

`DataTable.IsTheSameAs(other)` SHALL return true exactly when `IsEqualTo` is true and the table names, the column names in order and the row names in order are equal. A row without a name SHALL equal only a row without a name.

#### Scenario: A clone
- **WHEN** a table with a table name, column names and row names is cloned
- **THEN** `IsTheSameAs(clone)` is true

#### Scenario: One row name differs
- **WHEN** two tables differ only in the name of row 2
- **THEN** `IsEqualTo` is true and `IsTheSameAs` is false

### Requirement: Identity and nil

Both methods SHALL return true when given the receiver itself, and SHALL return false and log a warning when given nil.

#### Scenario: Nil argument
- **WHEN** `dt.IsEqualTo(nil)` is called
- **THEN** it returns false
