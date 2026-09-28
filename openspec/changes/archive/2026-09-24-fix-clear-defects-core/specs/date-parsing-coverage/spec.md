## ADDED Requirements

### Requirement: Every timestamp magnitude reads as the unit it is

整數時間戳 SHALL 依位數判讀為 Excel 序號、秒、毫秒、微秒或奈秒，各區間之間 SHALL NOT 留下空隙。16 到 18 位 SHALL 讀為微秒。

#### Scenario: A microsecond timestamp
- **WHEN** `ConvertToDateString(int64(1700000000000000), layout)`
- **THEN** 得到 2023 年的日期，而不是把它當秒算出的 53872 年
