# Proposal: quant-volatility-zigzag

## Why

[#471](https://github.com/HazelnutParadise/insyra/issues/471): a pattern study needs turning points whose reversal threshold follows the stock's volatility, and that restart at data breaks. `quant.ZigZag` took one fixed threshold and carried state across any gap in the data.

## What Changes

- `quant.ZigZagBy(prices, thresholds, opts)` is ZigZag with a threshold for every row, taken at the extreme being confirmed, so it stays causal. `ZigZagOptions.Breaks` restarts the search at each break, and `Pivot.Segment` says which segment a pivot is in.
- `quant.VolatilityThresholds(prices, opts)` returns `max(Floor, Multiple x standard deviation of the last Window daily log returns)` for every row, with a stand-in volatility while there are too few returns.
- `quant.ZigZag` returns what it did before; `Pivot` gains a `Segment` field, always 0 there.

## Capabilities

### New Capabilities

- `quant-volatility-zigzag`: turning points whose reversal threshold follows the volatility of the series and that restart at data breaks, so a pattern study can use them without looking ahead.

## Impact

`quant/`, tests, `Docs/quant.md`, both CHANGELOGs.
