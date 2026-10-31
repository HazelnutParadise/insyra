## ADDED Requirements

### Requirement: A stream delivers messages until it ends

`FugleStockStream` and `FugleFutOptStream` SHALL deliver every snapshot and data frame of their subscriptions on `Messages()` as a `FugleStreamMessage` with the channel, symbol, snapshot flag, time and Fugle's fields flattened with underscores, split a candles snapshot into one message per candle, and close `Messages()` exactly once when the stream ends.

#### Scenario: A trade

- **WHEN** Fugle sends a trades data frame for 2330 with time 1792560600000000
- **THEN** a message arrives with Channel trades, Symbol 2330, Snapshot false and Time equal to that many microseconds

### Requirement: A stream survives the drops Fugle says to retry

A stream SHALL reconnect with growing pauses after a lost connection, server maintenance or an authentication outage, authenticate again, renew every subscription and use the renewed ids to unsubscribe. It SHALL end, with `Err()` returning `ErrFugleAuth`, `ErrFuglePlan` or the server's message, after an invalid key, a plan without the product or the connection limit, and `Close` SHALL NOT clear that error.

#### Scenario: A dropped connection

- **WHEN** the server drops the connection without a close frame after a subscription
- **THEN** the stream reconnects, subscribes again, and later unsubscribes with the id the new connection gave

### Requirement: A slow reader does not stall the stream

When `Messages()` is full a stream SHALL drop the oldest queued message, count it in `Dropped()`, and keep reading the connection so that Fugle's pings are answered.

#### Scenario: More messages than the buffer

- **WHEN** 1,525 messages arrive and none is read
- **THEN** Dropped is 501 and the oldest message still queued is the 502nd
