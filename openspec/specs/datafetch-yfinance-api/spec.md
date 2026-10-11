# datafetch-yfinance-api Specification

## Purpose
定義 `datafetch` 的 Yahoo Finance API 中設定值的名稱：每個財報頻率只有一個名稱，舊的別名保留為 Deprecated。

## Requirements

### Requirement: One statement frequency has one name

`YFPeriodAnnual` SHALL be the name for annual statements. `YFPeriodYearly` SHALL keep working on the 0.3.x line as a Deprecated constant with its value `"yearly"`, whose doc comment names `YFPeriodAnnual`, so a table fetched with it keeps its `yearly` label; the constant is removed in 0.4.

#### Scenario: The deprecated synonym
- **WHEN** 讀取 `YFPeriodYearly` 的 doc comment 與值
- **THEN** 有指名 `YFPeriodAnnual` 的 `Deprecated:` 段落，值仍為 `"yearly"`
