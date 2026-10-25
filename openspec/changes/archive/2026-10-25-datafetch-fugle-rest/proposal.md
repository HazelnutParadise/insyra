# Proposal: datafetch-fugle-rest

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440): Insyra has no minute candles and no intraday quotes, and the official Go-less SDKs leave Taiwan users with Python or Node. Fugle's market data REST API works with a free key and no brokerage account. The owner chose, on 2026-10-25, to follow the Python SDK's scope and parameter names but Insyra's flat method style, and to build the endpoints a free key can verify first.

## What Changes

- New `datafetch.Fugle(FugleConfig)` returning a `FugleClient`, a thin client over the shared `TWStockClient` request machinery (limiter, retries, context).
- Methods, each with a `Context` form: `Stock.HistoricalCandles`, `Stock.IntradayCandles`, `Stock.IntradayQuote`, `Stock.IntradayTrades`, `Stock.IntradayVolumes`, `Stock.IntradayTickers`. Results are `*insyra.DataTable`.
- A historical range over a year is cut into 364-day pieces fetched in turn and joined in ascending order; a 404 piece adds nothing. Week and month candles are not cut and are refused beyond 364 days.
- Typed errors `ErrFugleAuth` (401), `ErrFuglePlan` (403) and `ErrFugleThrottled` (429); only throttling is retried.
- Not in this change: futures and options, snapshots, technical indicators, corporate actions, ownership (paid plans), the WebSocket stream.

## Capabilities

### New Capabilities

- `datafetch-fugle`: the Fugle REST client.

## Impact

`datafetch/fugle.go`, `datafetch/fugle_test.go`, `datafetch/fugle_live_test.go`, `Docs/datafetch.md`, both CHANGELOGs. Additive.
