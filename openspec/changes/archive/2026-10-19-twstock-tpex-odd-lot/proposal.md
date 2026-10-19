# Proposal: twstock-tpex-odd-lot

## Why

[#451](https://github.com/HazelnutParadise/insyra/issues/451) asked for the odd-lot reports of both exchanges. `TWStockClient.OddLotQuotes` covers the TWSE (intraday since 2020-10-26, after-hours since 2005). The TPEx half is not done.

## What Changes

- `TWStockClient.OddLotQuotes(date, session, market)` (and `...Context`) takes the market, `TWMarketTWSE`, `TWMarketTPEx` or `TWMarketAuto`, and so reads the TPEx intraday or after-hours odd-lot report of a trading day in the same columns as the TWSE's. It replaces `OddLotQuotes(date, session)`, which was not yet released.
- The doc comment of `OddLotQuotes`, which says the TPEx publishes only the latest day, is corrected.

## Capabilities

### New Capabilities

- `twstock-tpex-odd-lot`: the TPEx odd-lot reports by date.

## Impact

`datafetch/twstock_oddlot.go`, tests, `Docs/datafetch.md`, both CHANGELOGs.
