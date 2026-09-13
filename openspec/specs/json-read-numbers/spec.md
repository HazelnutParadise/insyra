# json-read-numbers Specification

## Purpose
Guarantees `ReadJSON_File` and `ReadJSON` type JSON numbers identically — integer literals as `int64`, decimals as `float64` — by sharing one decode path, so a value never changes type depending on whether it arrived as a file or as bytes.

## Requirements
### Requirement: ReadJSON_File types numbers the same way as ReadJSON

`ReadJSON_File` SHALL 以與 `ReadJSON` 相同的解碼路徑讀取檔案：整數字面值 SHALL 成為 `int64`（超過 2^53 不失真），小數字面值 SHALL 成為 `float64`。檔案內容為單一物件時 SHALL 載入為一列。

#### Scenario: A large integer keeps its value

- **WHEN** 檔案內容為 `[{"id": 9007199254740993}]`，呼叫 `ReadJSON_File`
- **THEN** 該格為 `int64(9007199254740993)`

#### Scenario: File and bytes agree

- **WHEN** 同一份 JSON 分別以 `ReadJSON_File(path)` 與 `ReadJSON(bytes)` 載入
- **THEN** 兩張表的每一格型別與值相同

#### Scenario: A single object loads as one row

- **WHEN** 檔案內容為 `{"a": 1}`
- **THEN** 回傳一列一欄的表

