## Why

Excess returns, Sharpe ratios and futures-plus-cash strategies need a Taiwan risk-free rate series, and `quant` takes one per period, but `datafetch` had no source for it, so callers typed in a constant (#690). The central bank publishes the interbank overnight call-loan rate daily from 1991 and the five major banks' posted deposit rates monthly from 2001 as open data.

## What Changes

- `datafetch.TWCentralBank(TWStockConfig)` creates a `TWCentralBankClient`. The name carries `TW` because "central bank" is a generic name; the bank's own abbreviation CBC is also used by other central banks.
- `OvernightRate(from, to)`: `Date`, `Rate`, `High`, `Low`, daily, percent per year, from `EG37D01.csv`; month summary rows are dropped and days without trades are nil.
- `DepositRate(from, to)`: `Month`, `Rate`, the mean of the five banks' one-year fixed-rate time deposit rate from `A13Rate.csv`, with the bank's own membership (彰化銀行 to 2008-10, 臺灣土地銀行 from 2008-11); a month without all five banks is nil.
- Both have Context forms; a zero `from` or `to` leaves that end open.

## Impact

- New `datafetch/twcentralbank.go`, `Docs/datafetch.md`, both changelogs. No existing behaviour changes.
