# Proposal: ta-trend-indicators

## Why

[#481](https://github.com/HazelnutParadise/insyra/issues/481), batch 2 of 5: the trend indicators.

## What Changes

- `ta.Aroon`, `ta.TRIX` (with a signal line), `ta.KAMA` and `ta.Ichimoku`.
- Ichimoku's cloud is aligned to the row it is in force on, with no projection ahead and no lagging line, after a design review.

## Capabilities

### New Capabilities

- `ta-trend-indicators`: Aroon, TRIX, KAMA and Ichimoku.

## Impact

`quant/ta/trend.go`, tests, `Docs/quant-ta.md`, both CHANGELOGs.
