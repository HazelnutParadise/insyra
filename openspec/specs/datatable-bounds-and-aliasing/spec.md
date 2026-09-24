# datatable-bounds-and-aliasing Specification

## Purpose
Guarantees that a `DataTable` never panics on a bad index (it sets `Err()`), that a `Filter*` call with no match returns a usable empty table, and that `FilterRows`/`FilterCols` handle ragged columns.

## Requirements
### Requirement: Bad indices set Err instead of panicking

`DataTable.GetElementByNumberIndex`、`SetRowToColNames`、`SetColToRowNames` 遇到不存在的欄或列 SHALL 設定 `Err()` 並回傳 nil／自身，SHALL NOT panic。`GetElementByNumberIndex` 的負欄索引 SHALL 由尾端往回數。

#### Scenario: Column number out of range
- **WHEN** 單欄表呼叫 `GetElementByNumberIndex(0, 5)`
- **THEN** 回傳 nil，`Err()` 非 nil

#### Scenario: Row index out of range for SetRowToColNames
- **WHEN** 兩列表呼叫 `SetRowToColNames(99)`
- **THEN** 表不變，`Err()` 非 nil

#### Scenario: Column index unknown for SetColToRowNames
- **WHEN** 單欄表呼叫 `SetColToRowNames("ZZ")`
- **THEN** 表不變，`Err()` 非 nil

### Requirement: Filter results are usable and tolerate ragged tables

`Filter*` 方法找不到符合項時 SHALL 回傳 `NewDataTable()`，其方法 SHALL 可安全呼叫；`FilterRows`／`FilterCols` SHALL 以整表列數處理，短欄以 nil 補，SHALL NOT panic。

#### Scenario: Methods on a not-found filter result do not panic
- **WHEN** `e := src.FilterColsByColNameEqualTo("zzz"); e.GetRowIndexByName("x"); e.SwapRowsByName("x", "y")`
- **THEN** 不 panic

#### Scenario: FilterRows on a ragged table
- **WHEN** 第一欄 3 列、第二欄 1 列的表呼叫 `FilterRows(func(...) bool { return true })`
- **THEN** 回傳 3 列，第二欄後兩列為 nil，不 panic
