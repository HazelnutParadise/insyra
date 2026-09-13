## ADDED Requirements

### Requirement: One numeric definition

`DropColsContainNumber`、`DropRowsContainNumber` SHALL 以 Go 內建整數與浮點型別判定（`int`、`int8`…`int64`、`uint`…`uint64`、`float32`、`float64`），`time.Duration` 等具名數值型別 SHALL NOT 算數值；`DataTable.Mean` SHALL 只以可轉成 float64 的格子數作分母，無數值格時回 NaN。

#### Scenario: int64 column is dropped
- **WHEN** 含 `int64` 欄與字串欄的表呼叫 `DropColsContainNumber()`
- **THEN** 只剩字串欄

#### Scenario: A Duration column is kept
- **WHEN** 含 `time.Duration` 欄與字串欄的表呼叫 `DropColsContainNumber()`
- **THEN** 兩欄都保留

#### Scenario: Mean ignores non-numeric cells in the denominator
- **WHEN** 表為 `[2, "x"]`、`[4, nil]`
- **THEN** `Mean()` 為 3
