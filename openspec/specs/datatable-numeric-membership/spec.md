# datatable-numeric-membership Specification

## Purpose
One definition of "numeric" — `IsNumeric` — for `DropColsContainNumber`, `DropRowsContainNumber` and the denominator of `DataTable.Mean`.

## Requirements
### Requirement: One numeric definition

`DropColsContainNumber`、`DropRowsContainNumber` SHALL 以 `IsNumeric` 判定（含 int64、float32 等所有 Go 數值型別）；`DataTable.Mean` SHALL 只以可轉成 float64 的格子數作分母，無數值格時回 NaN。

#### Scenario: int64 column is dropped
- **WHEN** 含 `int64` 欄與字串欄的表呼叫 `DropColsContainNumber()`
- **THEN** 只剩字串欄

#### Scenario: Mean ignores non-numeric cells in the denominator
- **WHEN** 表為 `[2, "x"]`、`[4, nil]`
- **THEN** `Mean()` 為 3

