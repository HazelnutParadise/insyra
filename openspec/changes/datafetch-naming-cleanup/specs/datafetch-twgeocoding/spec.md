## ADDED Requirements

### Requirement: ReverseTable takes column selectors

`ReverseTable(dt, latCol, lngCol any)` SHALL resolve each column with the library's column selector: a string is an Excel-style index, `insyra.Name(...)` is a name compared exactly, an int is a 0-based position counting from the end when negative. A selector that does not resolve SHALL return an error naming that selector and whether it was the latitude or the longitude column, before any request. `ReverseTableByColName` SHALL remain for one release as a Deprecated method that calls `ReverseTable` with `insyra.Name` of each argument.

#### Scenario: Three spellings of the same columns
- **WHEN** 表格有名為 `lat`、`lng` 的 A、B 兩欄，分別以 `"A", "B"`、`insyra.Name("lat"), insyra.Name("lng")`、`0, 1` 呼叫 `ReverseTable`
- **THEN** 三次都解析同樣的兩欄，結果相同

#### Scenario: A bare string is an index, not a name
- **WHEN** 以 `"lat", "lng"` 呼叫 `ReverseTable`
- **THEN** 回傳指出緯度欄選擇器 `"lat"` 無法解析的錯誤，且沒有發出請求

#### Scenario: The deprecated method keeps its meaning
- **WHEN** 以 `"lat", "lng"` 呼叫 `ReverseTableByColName`
- **THEN** 結果與 `ReverseTable(dt, insyra.Name("lat"), insyra.Name("lng"))` 相同
