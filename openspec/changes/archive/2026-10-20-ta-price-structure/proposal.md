# Proposal: ta-price-structure

## Why

[#481](https://github.com/HazelnutParadise/insyra/issues/481), batch 5a of 5: the price-structure tools. Candlestick patterns follow in their own change.

## What Changes

- `ta.PivotPoints` (floor-trader levels; the type for ZigZag's turning points is `Swing`), `ta.HeikinAshi`, `ta.TDSequential` and `ta.PointAndFigure`.
- None has a TA-Lib reference; they are tested with hand-built series, and the documentation states the rules, including that TD Sequential's finer rules differ between platforms.

## Capabilities

### New Capabilities

- `ta-price-structure`: the tools above.

## Impact

`quant/ta/structure.go`, tests, `Docs/quant-ta.md`, both CHANGELOGs.
