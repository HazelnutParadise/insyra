# Proposal: datafetch-fugle-stats-snapshot

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440), stage 4a: the first of four changes for the remaining stock endpoints, which the owner approved on 2026-10-30 ("do all the Fugle stock endpoints") after a Fable review. This one adds the historical statistics and the three snapshots and gives the market a type.

## What Changes

- `client.Stock.HistoricalStats(symbol)` returns one row of price statistics.
- `client.Stock.SnapshotQuotes`, `SnapshotMovers` and `SnapshotActives` return a market as of now, its biggest movers and its most active stocks, each with the snapshot's date, time and market on every row.
- New string-valued types `FugleMarket`, `FugleMoverDirection`, `FugleChangeBasis`, `FugleTradeBasis`, `FugleThresholdOp` and the struct `FugleThreshold{Op, Value}` (the zero value keeps every stock); required API choices are positional parameters, `OnlyCommonStock` and the threshold are options.
- `FugleIntradayTickersOptions.Market` becomes a `FugleMarket`, checked before a request is sent (the client is unreleased, so no caller breaks).
- Not in this change: technical indicators, corporate actions and ownership, which follow as their own changes.

## Capabilities

### Modified Capabilities

- `datafetch-fugle`: the statistics and snapshot methods and the market type.

## Impact

`datafetch/fugle.go`, `datafetch/fugle_stock_snapshot.go`, `datafetch/fugle_test.go`, `Docs/datafetch.md`, both CHANGELOGs. Additive apart from the unreleased `Market` field type. The snapshots need a paid plan, so they are tested against a local server only.
