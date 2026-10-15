# datatable-equality Specification

## Purpose
How two `DataTable` values are compared. `IsEqualTo` compares the shape and every cell and ignores names, and `IsTheSameAs` also requires the table, column and row names to match.

## Requirements

### Requirement: IsEqualTo compares shape and cells

`DataTable.IsEqualTo(other)` SHALL return true exactly when both tables have the same number of rows and columns and every cell is equal under `DataList.IsEqualTo`'s rule. A cell past the end of a short column SHALL count as `nil`. Table, column and row names SHALL NOT be compared.

#### Scenario: Same cells, different names
- **WHEN** 兩張表格子都是 `[[1, 2], ["a", "b"]]`，但表名、欄名、列名都不同
- **THEN** `IsEqualTo` 為 true，`IsTheSameAs` 為 false

#### Scenario: A cell of another type
- **WHEN** 兩張表只差一格，一邊是 `int(1)`，另一邊是 `int64(1)`
- **THEN** `IsEqualTo` 為 false

#### Scenario: NaN cells
- **WHEN** 兩張表同一格都是 `math.NaN()`，其他格相同
- **THEN** `IsEqualTo` 為 true

#### Scenario: Different shape
- **WHEN** 一張表多一列或多一欄
- **THEN** `IsEqualTo` 為 false

### Requirement: IsTheSameAs adds every name

`DataTable.IsTheSameAs(other)` SHALL return true exactly when `IsEqualTo` is true and the table names, the column names in order and the row names in order are equal. A row without a name SHALL equal only a row without a name.

#### Scenario: A clone
- **WHEN** 對有表名、欄名、列名的表呼叫 `Clone()`
- **THEN** `IsTheSameAs(clone)` 為 true

#### Scenario: One row name differs
- **WHEN** 兩張表只有第 2 列的列名不同
- **THEN** `IsEqualTo` 為 true，`IsTheSameAs` 為 false

### Requirement: Identity and nil

Both methods SHALL return true when given the receiver itself, and SHALL return false and log a warning, without recording an error, when given nil.

#### Scenario: Nil argument
- **WHEN** 呼叫 `dt.IsEqualTo(nil)`
- **THEN** 回傳 false，`dt.Err()` 為 nil
