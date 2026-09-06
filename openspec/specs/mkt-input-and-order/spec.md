# mkt-input-and-order Specification

## Purpose
`RFM` skips (with a warning) rather than panics on a non-numeric amount, and `RFM`/`CustomerActivityIndex` emit rows in sorted customer-ID order so runs are reproducible.

## Requirements
### Requirement: RFM does not panic on input

`RFM` 遇到無法轉成數值的金額 SHALL 記錄警告並跳過該列，SHALL NOT panic。

#### Scenario: Text amount
- **WHEN** 金額欄含 `"abc"`
- **THEN** `RFM` 回傳表，該列不計入

### Requirement: Output order is deterministic

`RFM` 與 `CustomerActivityIndex` 的輸出列 SHALL 依 CustomerID 字典序排列。

#### Scenario: Two runs agree
- **WHEN** 同一輸入執行 `RFM` 兩次
- **THEN** 兩張表的 CustomerID 欄逐列相同且已排序

