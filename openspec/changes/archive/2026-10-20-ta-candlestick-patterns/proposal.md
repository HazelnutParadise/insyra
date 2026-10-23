# Proposal: ta-candlestick-patterns

## Why

[#481](https://github.com/HazelnutParadise/insyra/issues/481), the last change: candlestick pattern flags with configurable body and shadow ratios.

## What Changes

- `ta.CandlePatterns` returns one named list per pattern, +1 bullish, -1 bearish, 0 none, flagged on the row that completes it. Thirteen patterns to start; adding one adds a field.
- One function with a result struct, and not one function per pattern, because all the patterns share the same bar geometry and the same options.

## Capabilities

### New Capabilities

- `ta-candlestick-patterns`: candlestick pattern flags.

## Impact

`quant/ta/candles.go`, tests, `Docs/quant-ta.md`, both CHANGELOGs.
