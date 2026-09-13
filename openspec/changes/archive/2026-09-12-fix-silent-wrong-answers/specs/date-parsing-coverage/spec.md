## ADDED Requirements

### Requirement: A millisecond timestamp reads as the right date across its whole range

以毫秒解讀時間戳的程式碼 SHALL NOT 先把毫秒乘成奈秒再交給 `time.Unix`，該乘法在超過 9223372036854 毫秒（約 2262-04-11）時會溢位 `int64` 並得到另一個日期。該分支接受的整個區間 SHALL 都轉出正確的日期。

#### Scenario: A far-future millisecond timestamp
- **WHEN** `ConvertToDateString(int64(99999999999999), layout)`
- **THEN** 得到 5138 年的日期，而不是溢位後的 2216 年
