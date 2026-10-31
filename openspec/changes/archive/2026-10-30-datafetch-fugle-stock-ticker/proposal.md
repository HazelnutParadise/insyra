# Proposal: datafetch-fugle-stock-ticker

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440): the stock client reads every intraday endpoint except the one-symbol ticker, `stock/intraday/ticker/{symbol}`, which the futures and options client already has as `FutOpt.IntradayTicker`. It was found missing while comparing against Fugle's official Go SDK, inside the owner's 2026-10-30 approval to do all the Fugle stock endpoints.

## What Changes

- `client.Stock.IntradayTicker(symbol, oddLot...)` and its Context form return one row of a stock's details for the day: name, industry, security type, reference price, price limits, trading flags, status, board lot and currency.
- The documented columns are always present, nil where the API leaves a field out; fields only warrants or indices have follow when present.
