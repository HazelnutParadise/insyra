# Proposal: quant-chart-patterns

## Why

[#471](https://github.com/HazelnutParadise/insyra/issues/471): testing chart-pattern theory over every listed stock needed the patterns found from causal turning points, each with its formation, breakout and expiry, and the fan principle's three breaks. They were written in Python outside insyra.

## What Changes

- `quant.DetectPatterns(prices, pivots, opts)` finds head and shoulders tops and bottoms, double tops and bottoms, boxes, ascending, descending and symmetric triangles, rising and falling wedges, and diamonds. Each `Pattern` has its kind, start, the row that confirmed it, its expiry, its breakout row and direction, and the trend before it. Tolerances are `PatternOptions`.
- `quant.FanBreaks(prices, pivots, opts)` returns the closes through fan lines, numbered 1 to 3.
- `PatternKind`, `BreakDirection` with `String()`.

## Capabilities

### New Capabilities

- `quant-chart-patterns`: chart patterns and fan-principle breaks found from causal turning points, each reported only from the row on which it could be known, for backtesting price-pattern theory.

## Impact

`quant/`, tests, `Docs/quant.md`, both CHANGELOGs.
