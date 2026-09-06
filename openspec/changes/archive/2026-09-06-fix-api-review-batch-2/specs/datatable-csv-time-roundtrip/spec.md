## ADDED Requirements

### Requirement: ToCSV writes times ParseDates can read

`ToCSV` 遇到 `time.Time` 格子 SHALL 以 RFC 3339（含奈秒）寫出；經 `ReadCSV_File` 與 `ParseDates` 讀回 SHALL 得到相同時刻。

#### Scenario: Round trip
- **WHEN** 含 `time.Time` 欄的表 `ToCSV` 後以 `ReadCSV_FileWithOptions`（RawStrings）讀回並對該欄 `ParseDates()`
- **THEN** 每格 `Equal` 原時刻
