# Proposal: datafetch-fugle-stream

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440): the REST endpoints are done; Fugle's live data comes over a WebSocket stream. The owner approved the design on 2026-10-31 after a Fable review. Fugle's own Go SDK needs cgo and hands back raw JSON, so a pure-Go stream that ends in a DataTable is what brings Fugle's users to Insyra.

## What Changes

- `client.Stock.Stream()` / `StreamContext(ctx)` and `client.FutOpt.Stream()` / `StreamContext(ctx)` connect and authenticate, returning `*FugleStockStream` and `*FugleFutOptStream`.
- Each stream has `Subscribe(channel, symbols, flag...)`, `Unsubscribe`, `Messages() <-chan FugleStreamMessage`, `Collect(ctx, channel, n)`, `Dropped()`, `Err()` and `Close()`; the flag is `oddLot` for stocks and `afterHours` for futures and options, as in the REST methods.
- `FugleChannel` names Fugle's channels; `FugleStreamMessage` carries the channel, symbol, snapshot flag, time and Fugle's fields flattened as the REST tables name them.
- The stream answers pings, reconnects and resubscribes after the drops Fugle says to retry, ends with `ErrFugleAuth`, `ErrFuglePlan` or the server's message otherwise, and drops the oldest messages instead of blocking when they are read too slowly.
- `github.com/gorilla/websocket` (v1.5.3, outside every advisory) becomes a direct dependency.
