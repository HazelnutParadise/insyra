# datatable-find-hygiene Specification

## Purpose
`DataTable` 查找與建表的衛生規則：`FindColsIfContains*` 遇到不含該值的欄不當成錯誤記錄；由 map 建立欄位時依欄名排序，同一輸入每次得到相同欄序。

## Requirements
### Requirement: Find helpers do not record non-matches as errors

`FindColsIfContains`／`FindColsIfContainsAll` 對不含該值的欄 SHALL NOT 設定 `Err()`，也 SHALL NOT 在錯誤緩衝區新增記錄。

#### Scenario: Value absent from some columns
- **WHEN** 兩欄表只有一欄含 5，清空錯誤緩衝區後呼叫 `FindColsIfContains(5)`
- **THEN** 回傳一欄，`Err()` 為 nil，`GetErrorCount()` 為 0

### Requirement: Row maps add columns in a deterministic order

`AppendRowsByColName`（以及經它建表的 `ReadJSON`／`ReadJSON_File`）從一個 map 新增多個欄位時，SHALL 依欄名排序後新增，使同一輸入每次產生相同欄序；已存在的欄保持原位置。

#### Scenario: Two new keys in one row
- **WHEN** 對空表呼叫 `AppendRowsByColName(map{"b":1,"a":2})` 一百次各建新表
- **THEN** 每張表的欄名順序都是 `a, b`
