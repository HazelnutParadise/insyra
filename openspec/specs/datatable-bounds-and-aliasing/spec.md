# datatable-bounds-and-aliasing Specification

## Purpose
Guarantees that a `DataTable` never panics on a bad index (it sets `Err()`) and never hands out or shares its own storage: `Data`/`ToMap` return copies and every `Filter*` result owns its columns and row names.

## Requirements
### Requirement: Bad indices set Err instead of panicking

`DataTable.GetElementByNumberIndex`、`SetRowToColNames`、`SetColToRowNames` 遇到不存在的欄或列 SHALL 設定 `Err()` 並回傳 nil／自身，SHALL NOT panic。

#### Scenario: Column number out of range
- **WHEN** 單欄表呼叫 `GetElementByNumberIndex(0, 5)`
- **THEN** 回傳 nil，`Err()` 非 nil

#### Scenario: Row index out of range for SetRowToColNames
- **WHEN** 兩列表呼叫 `SetRowToColNames(99)`
- **THEN** 表不變，`Err()` 非 nil

#### Scenario: Column index unknown for SetColToRowNames
- **WHEN** 單欄表呼叫 `SetColToRowNames("ZZ")`
- **THEN** 表不變，`Err()` 非 nil

### Requirement: Data and filtered tables never alias table storage

`Data()`／`ToMap()` 回傳的每個 slice SHALL 是複本；所有 `Filter*` 方法回傳的表 SHALL 擁有自己的欄位資料與列名索引；找不到時 SHALL 回 `NewDataTable()`；`FilterRows`／`FilterCols` SHALL 以整表列數處理，短欄以 nil 補。

#### Scenario: Mutating Data output leaves the table unchanged
- **WHEN** `m := dt.Data(); m["a"][0] = 99`
- **THEN** `dt.GetElement(0, "A")` 仍是原值

#### Scenario: Editing a filtered table leaves the source unchanged
- **WHEN** `f := src.FilterColsByColNameEqualTo("a"); f.UpdateElement(0, "A", 99)`
- **THEN** `src.GetElement(0, "A")` 仍是原值

#### Scenario: Methods on a not-found filter result do not panic
- **WHEN** `e := src.FilterColsByColNameEqualTo("zzz"); e.GetRowIndexByName("x"); e.SwapRowsByName("x", "y")`
- **THEN** 不 panic

#### Scenario: FilterRows on a ragged table
- **WHEN** 第一欄 3 列、第二欄 1 列的表呼叫 `FilterRows(func(...) bool { return true })`
- **THEN** 回傳 3 列，第二欄後兩列為 nil，不 panic

