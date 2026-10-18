## MODIFIED Requirements

### Requirement: Row maps add columns in a deterministic order

`AppendRowsByColName`（以及經它建表的 `ReadJSON`／`ReadJSONFile`）從一個 map 新增多個欄位時，SHALL 依欄名排序後新增，使同一輸入每次產生相同欄序；已存在的欄保持原位置。

#### Scenario: Two new keys in one row
- **WHEN** 對空表呼叫 `AppendRowsByColName(map{"b":1,"a":2})` 一百次各建新表
- **THEN** 每張表的欄名順序都是 `a, b`
