# Proposal: datafetch-fugle-futopt-intraday

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440), stage 2: after the stock endpoints, Fugle's futures and options intraday data. The owner approved on 2026-10-25, after a Fable review, building the seven intraday endpoints first and the two historical ones as a following change.

## What Changes

- `FugleClient.FutOpt` (a `FugleFutOpt`) with `IntradayProducts`, `IntradayTickers`, `IntradayTicker`, `IntradayQuote`, `IntradayCandles`, `IntradayTrades` and `IntradayVolumes`, each with a `Context` form, returning `*insyra.DataTable`.
- `FugleFutOptType` (`FugleFuture`, `FugleOption`) for the kind of product, a required parameter of the list methods.
- Night-session data is one setting (`AfterHours`, or the trailing `afterHours` flag); the client sends the value the endpoint wants (`AFTERHOURS` for products and tickers, `afterhours` for the rest).
- A spread symbol such as `TXFJ6/K6` is accepted (one slash, both parts non-empty); a stock symbol still refuses a slash.
- A stepped price limit in a ticker becomes numbered columns; a quote's nested objects are joined with underscores.
- Intraday candles refuse a timeframe other than 1, 5, 10, 15, 30 and 60 minutes.
- Not in this change: the historical endpoints, which come next.

## Capabilities

### Modified Capabilities

- `datafetch-fugle`: the futures and options intraday methods.

## Impact

`datafetch/fugle.go` (the `FutOpt` field), `datafetch/fugle_futopt.go`, `datafetch/fugle_futopt_test.go`, `Docs/datafetch.md`, both CHANGELOGs. Additive. The endpoints are not on the free plan, so they are tested against a local server only.
