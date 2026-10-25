# Proposal: datafetch-fugle-optional-settings

## Why

Codex's review of the 0.4 port of `datafetch-fugle-rest` (#568) found two gaps against the repository's conventions in the client that #566 merged on dev and that has not been released: settings that all have defaults were a required `...Params` struct, and a column the API omits from every row disappeared from the table although the documentation promises nil cells.

## What Changes

- `HistoricalCandles`, `IntradayCandles`, `IntradayTrades` and `IntradayTickers` take their settings as one optional trailing `...Options` value (`FugleHistoricalCandlesOptions` and so on, renamed from `...Params`); more than one value is an error. `IntradayTickers` takes the ticker type as its first parameter, since it has no default.
- The tickers, trades and volumes tables always carry their documented columns (`industry`; `bid`, `ask`, `volume`; the at-bid and at-ask volumes), as nil cells where the API leaves them out.

## Capabilities

### Modified Capabilities

- `datafetch-fugle`: optional settings and stable columns.

## Impact

`datafetch/fugle.go`, `datafetch/fugle_test.go`, `datafetch/fugle_live_test.go`, `Docs/datafetch.md`. Breaking only against the unreleased signatures of #566.
