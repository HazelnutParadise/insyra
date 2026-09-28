## ADDED Requirements

### Requirement: ReadJSON accepts only JSON

`ReadJSON` and `ReadJSON_File` SHALL refuse input that is not valid JSON under RFC 8259, returning an error and no table.

#### Scenario: A leading zero is refused

- **WHEN** `ReadJSON` 讀 `[{"a":01}]`
- **THEN** 回傳錯誤，表為 nil

#### Scenario: A trailing comma is refused

- **WHEN** `ReadJSON` 讀 `[{"a":1,}]`
- **THEN** 回傳錯誤，表為 nil

### Requirement: A number outside float64's range keeps its text

`ReadJSON` SHALL read a JSON number that is neither an `int64` nor a finite `float64` as the string it was written as, the way `ReadCSV` reads the same text.

#### Scenario: 1e400

- **WHEN** `ReadJSON` 讀 `[{"a":1e400,"b":1}]`
- **THEN** 讀取成功，`a` 那格是字串 `"1e400"`，`b` 那格是 `int64(1)`

### Requirement: Invalid UTF-8 is replaced as encoding/json replaces it

`ReadJSON` SHALL replace each invalid UTF-8 byte inside a JSON string with U+FFFD, as `encoding/json` does.

#### Scenario: A stray 0xFF byte

- **WHEN** `ReadJSON` 讀的字串值裡有一個 0xFF 位元組
- **THEN** 那格的字串在該位置是 U+FFFD，與 `encoding/json` 解碼同一份輸入的結果相同

### Requirement: ToJSON writes what encoding/json writes

`ToJSON_Bytes` SHALL return, byte for byte, what `encoding/json.MarshalIndent` returns with a two-space indent for the same rows, and `ToJSON` and `ToJSON_String` SHALL write the same bytes.

#### Scenario: Small exponents, escapes and mixed cells

- **WHEN** 一張表含 `1e-7`、`2.5e-8`、`1e21`、`int64` 極值、`<a&b>`、U+2028、中文、`nil`、`bool`、`time.Time` 與 `float32(0.1)`
- **THEN** `ToJSON_Bytes(true)` 與 `ToJSON_Bytes(false)` 都與 `encoding/json.MarshalIndent` 對同樣的列輸出逐位元組相同，其中 `1e-7` 寫成 `1e-7`
