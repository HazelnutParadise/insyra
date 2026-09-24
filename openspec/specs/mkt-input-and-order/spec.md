# mkt-input-and-order Specification

## Purpose
`RFM` skips (with a warning) rather than panics on an amount it cannot read, while still reading numeric strings as numbers, and `RFM`/`CustomerActivityIndex` emit rows in sorted customer-ID order so runs are reproducible.

## Requirements
### Requirement: RFM does not panic on input

`RFM` SHALL 以 `conv.ParseF64` 相同的規則讀金額：Go 數值型別直接讀取，字串去除前後空白後解析。無法讀成數值的金額 SHALL 記錄指出列號的警告並跳過該列，SHALL NOT panic。

#### Scenario: Text amount
- **WHEN** 金額欄含 `"abc"`
- **THEN** `RFM` 回傳表，該列不計入

#### Scenario: Numeric string amount
- **WHEN** 金額欄為 `"10"`、`" 20.5 "`，與以數值 `10`、`20.5` 呼叫比較
- **THEN** 兩次 `RFM` 的輸出表相同

### Requirement: Output order is deterministic

`RFM` 與 `CustomerActivityIndex` 的輸出列 SHALL 依 CustomerID 字典序排列。

#### Scenario: Two runs agree
- **WHEN** 同一輸入執行 `RFM` 兩次
- **THEN** 兩張表的 CustomerID 欄逐列相同且已排序
