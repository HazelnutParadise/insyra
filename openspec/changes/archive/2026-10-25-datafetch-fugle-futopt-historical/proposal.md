# Proposal: datafetch-fugle-futopt-historical

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440), stage 3: the historical futures and options endpoints, the second half of the split the owner approved on 2026-10-25 after a Fable review ("intraday first, then the two historical ones").

## What Changes

- `client.FutOpt.HistoricalCandles(product, opts...)` returns the candles of a product's contract, a continuous contract or an option, with the day and minute ranges over 364 days cut into pieces as for stocks (the cutting is now shared code). Rows are ascending by date and contract month, a date with two contracts has two rows, and `contractMonth` is a column.
- `client.FutOpt.HistoricalDaily(product, opts...)` returns one trading day with a row per contract.
- `FugleCallPut` (`FugleCall`, `FuglePut`) and the options structs `FugleFutOptHistoricalCandlesOptions` and `FugleFutOptHistoricalDailyOptions`.
- Client-side checks the documentation states: a product without a slash, Strike and CallPut together and valid, a timeframe of 1, 5, 10, 15, 30, 60, D, W or M, fields from the documented set (`turnover` is refused), a week or month range under a year.
- A day or range with no data (HTTP 404) gives an empty table.

## Capabilities

### Modified Capabilities

- `datafetch-fugle`: the futures and options historical methods.

## Impact

`datafetch/fugle.go` (shared range cutting), `datafetch/fugle_futopt_historical.go`, `datafetch/fugle_futopt_test.go`, `Docs/datafetch.md`, both CHANGELOGs. Additive. Not testable with a free key; local-server tests only.
