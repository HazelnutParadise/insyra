# Proposal: twstock-tpex-odd-lot

## Why

[#451](https://github.com/HazelnutParadise/insyra/issues/451) asked for the odd-lot reports of both exchanges. `TWStockClient.OddLotQuotes` covers the TWSE (intraday since 2020-10-26, after-hours since 2005). The TPEx half is not done.

## What Changes

- `TWStockClient.TPExOddLotQuotesLatest()` (and `...Context`) returns the TPEx after-hours odd-lot report of the latest trading day, in the same columns as `OddLotQuotes(date, TWOddLotAfterHours)`.
- No history and no TPEx intraday odd lot: see design.

## Capabilities

### New Capabilities

- `twstock-tpex-odd-lot`: the latest day's TPEx odd-lot report.

## Impact

`datafetch/twstock_oddlot.go`, tests, `Docs/datafetch.md`, both CHANGELOGs.
