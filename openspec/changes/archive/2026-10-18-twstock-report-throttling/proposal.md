# Proposal: twstock-report-throttling

## Why

[#434](https://github.com/HazelnutParadise/insyra/issues/434): the TWSE site intermittently answers HTTP 307 with a "security" HTML page and no `Location` (one URL gave 307, 307, 200 at 3-second spacing). `TWStock` reported it only as `HTTP status 307`, which reads like a program error, and the documentation gave no values for backfilling years of history. The default `Retries` of 0 ends a backfill at the first block.

## What Changes

- HTTP 307 and 429 are reported with a message saying the source refused the request as too frequent and to raise `Interval` and `Retries`, and match the new exported `ErrTWStockThrottled` under `errors.Is`. They are retried exactly as before.
- `Docs/datafetch.md` gets a "Being throttled" paragraph with starting values.
- Not changed: the defaults of `Retries`, `RetryBackoff` and `Interval`. Changing them alters existing callers' behaviour and is the maintainer's decision.

## Capabilities

### Modified Capabilities

- `datafetch-twstock`: a throttled answer is named as such.

## Impact

`datafetch/twstock.go`, `datafetch/twstock_throttled_test.go`, `Docs/datafetch.md`, both CHANGELOGs.
