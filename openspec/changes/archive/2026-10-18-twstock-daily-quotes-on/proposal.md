# Proposal: twstock-daily-quotes-on

## Why

[#447](https://github.com/HazelnutParadise/insyra/issues/447): `AllDailyQuotes` returns only the latest day, so a market-breadth study, an equal-weighted market return or any whole-market study over history had no source, and the official advance and decline counts were out of reach.

## What Changes

- `TWStockClient.DailyQuotesOn(date, market)` reads any day's closes of every security from the TWSE `MI_INDEX` report (since 2004-02-11, delisted securities included) or the TPEx daily report, with a signed `Change` and the exchange's `Sign`.
- `TWStockClient.AdvanceDecline(date, market, commonStockOnly)` returns the official 漲跌證券數合計 counts when the TWSE published them, and otherwise counts from the closes' marks.
- The TWSE tables are found by title, since their positions differ between years.
- Checked live: the computed common-stock counts for 2026-10-16 (684 up, 286 down) equal the official stock column, and 2012-01-04 and the TPEx parse. The TPEx has no official table, so its counts are always computed.

## Capabilities

### New Capabilities

- `twstock-daily-quotes-on`: any day's whole-market closes and the advance and decline counts.

## Impact

`datafetch/twstock_daily_on.go`, `datafetch/twstock_parse.go` (the table title is read), fixtures, tests, `Docs/datafetch.md`, both CHANGELOGs.
