## 1. Tests

- [x] 1.1 Overnight rates: columns, nil rates on a day without trades, a month summary row dropped, the range kept
- [x] 1.2 Deposit rate: the five-bank mean with 彰化銀行 to October 2008 and 臺灣土地銀行 from November 2008, the month of the change, and nil for a month with fewer banks
- [x] 1.3 `to` before `from` sends nothing; changed headers, a bad month, a cancelled context and a zero client are errors

## 2. Implementation

- [x] 2.1 `TWCentralBank`, `OvernightRate`, `DepositRate` and their Context forms

## 3. Documentation

- [x] 3.1 `Docs/datafetch.md` section with the per-period conversion for `quant`, and both changelogs
