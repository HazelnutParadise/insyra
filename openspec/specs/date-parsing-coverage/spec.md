# date-parsing-coverage Specification

## Purpose
常見時間字串版面的辨識範圍，以及不得誤判非日期。
## Requirements
### Requirement: Common timestamp layouts parse

`TryParseTime` SHALL 接受 `2006-01-02 15:04:05`、`2006-01-02T15:04:05`、`2006-01-02 15:04` 與以 `/` 分隔的等價寫法；無時區的版面 SHALL 視為 UTC。非日期字串（純數字、單字、空字串）SHALL NOT 被視為日期。

#### Scenario: Zone-less timestamp
- **WHEN** 解析 `"2024-01-02 03:04:05"`
- **THEN** 成功，且等於 2024-01-02T03:04:05Z

### Requirement: A millisecond timestamp reads as the right date across its whole range

以毫秒解讀時間戳的程式碼 SHALL NOT 先把毫秒乘成奈秒再交給 `time.Unix`，該乘法在超過 9223372036854 毫秒（約 2262-04-11）時會溢位 `int64` 並得到另一個日期。該分支接受的整個區間 SHALL 都轉出正確的日期。

#### Scenario: A far-future millisecond timestamp
- **WHEN** `ConvertToDateString(int64(99999999999999), layout)`
- **THEN** 得到 5138 年的日期，而不是溢位後的 2216 年

### Requirement: Every timestamp magnitude reads as the unit it is

整數時間戳 SHALL 依位數判讀為 Excel 序號、秒、毫秒、微秒或奈秒，各區間之間 SHALL NOT 留下空隙。16 到 18 位 SHALL 讀為微秒。

#### Scenario: A microsecond timestamp
- **WHEN** `ConvertToDateString(int64(1700000000000000), layout)`
- **THEN** 得到 2023 年的日期，而不是把它當秒算出的 53872 年

### Requirement: A format pattern does not rewrite letters inside words

`ConvertDateFormat` SHALL 以掃描的方式處理樣式：同一個字母的連續段落視為一個 token，`[...]` 內的文字原樣輸出，其餘字元原樣通過。SHALL NOT 以逐個 ReplaceAll 掃過整個字串。

#### Scenario: A three-letter month
- **WHEN** 樣式是 `MMM DD`
- **THEN** 得到 `Jan 02`，而不是 `011 02`

#### Scenario: Literal text
- **WHEN** 樣式是 `[Date]: YYYY`
- **THEN** 得到 `Date: 2006`

