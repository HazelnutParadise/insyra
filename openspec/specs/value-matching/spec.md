# value-matching Specification

## Purpose
Defines how the library decides that a cell holds the value being looked for: integers match by value across Go integer types, a float never matches an integer, NaN matches NaN, encoder categories follow the same integer rule, and whole-list comparison stays type-strict.

## Requirements

### Requirement: Integers match by value across Go integer types

When the library looks for a value in a DataList or DataTable, whether counting, finding, replacing or dropping by value, a cell holding an integer SHALL match a searched integer of equal value regardless of their Go integer types. A negative signed integer SHALL NOT match any unsigned integer. A floating-point value SHALL NOT match an integer. A float64 NaN SHALL match a float64 NaN.

#### Scenario: Counting a CSV column with a Go literal
- **WHEN** 對從 CSV 讀入的整數欄（int64）呼叫 `Count(2)`
- **THEN** 回傳欄中 2 的個數

#### Scenario: A float is not an integer
- **WHEN** 對存有 `2.0` 的欄呼叫 `Count(2)`
- **THEN** 回傳 0

#### Scenario: Replacing from the CLI
- **WHEN** 在 CLI 對整數欄是 int64 的表執行 `replace t 2 0`
- **THEN** 原本的 2 都變成 0

### Requirement: Encoder categories compare integers by value

The one-hot, label and ordinal encoders SHALL treat integer categories of equal value as the same category, whatever their Go integer types.

#### Scenario: Ordinal order written as literals
- **WHEN** 以 `Order: []any{1, 2, 3}` 對 int64 欄做 `OrdinalEncode`
- **THEN** 每一格都得到對應的序號，不是 nil，也不是錯誤

#### Scenario: Fit on int, transform int64
- **WHEN** `LabelEncoder` 在 int 資料上 fit，再 Transform int64 資料
- **THEN** 同值的整數得到相同的編碼

### Requirement: Whole-list comparison stays type-strict

`IsEqualTo` and `IsTheSameAs` SHALL keep treating cells of different Go types as different.

#### Scenario: int versus int64 lists
- **WHEN** 呼叫 `NewDataList(1).IsEqualTo(NewDataList(int64(1)))`
- **THEN** 回傳 false
