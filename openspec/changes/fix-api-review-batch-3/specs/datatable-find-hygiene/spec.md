## ADDED Requirements

### Requirement: Find helpers do not record non-matches as errors

`FindColsIfContains`／`FindColsIfContainsAll` 對不含該值的欄 SHALL NOT 設定 `Err()`。

#### Scenario: Value absent from some columns
- **WHEN** 兩欄表只有一欄含 5，呼叫 `FindColsIfContains(5)`
- **THEN** 回傳一欄，`Err()` 為 nil

### Requirement: Row maps add columns in a deterministic order

`AppendRowsByColName`（以及經它建表的 `ReadJSON`／`ReadJSON_File`）從一個 map 新增多個欄位時，SHALL 依欄名排序後新增，使同一輸入每次產生相同欄序；已存在的欄保持原位置。

#### Scenario: Two new keys in one row
- **WHEN** 對空表呼叫 `AppendRowsByColName(map{"b":1,"a":2})` 一百次各建新表
- **THEN** 每張表的欄名順序都是 `a, b`
