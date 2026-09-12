## ADDED Requirements

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
