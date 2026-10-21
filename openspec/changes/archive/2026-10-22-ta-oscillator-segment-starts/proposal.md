# Proposal: ta-oscillator-segment-starts

## Why

`ta.StochRSI` and `ta.CR` carry their windows and smoothing across a trading halt or a relisting, as `ta.KD` did before `KDOptions.SegmentStarts`, so rows after the break mix two histories. `ta.StochRSI` also stayed `NaN` for good when the series began with a flat stretch, because its first undefined RSI windows entered the recursive smoothing. And its default smoothing differs from TA-Lib's and TradingView's without the documentation saying how to get theirs.

## What Changes

- `ta.StochRSI` honours `KDOptions.SegmentStarts`: each segment is computed on its own, the inner RSI included, for both smoothing methods.
- `CROptions` gains `SegmentStarts []int` with the same meaning; a segment shows NaN through its first n rows.
- Recursive smoothing in `ta.StochRSI` skips rows whose raw stochastic value is undefined and begins at the first row that has one.
- `Docs/quant-ta.md` states that StochRSI's default numbers differ from TA-Lib's and TradingView's and which `KDOptions` reproduce them.
- The zero value of every option changes nothing. The helpers `cleanSegmentStarts` and `segmentBounds` are reused.

## Capabilities

### Modified Capabilities

- `ta-oscillators`: StochRSI and CR restart at each segment start; StochRSI recovers from an undefined RSI.

## Impact

`quant/ta/oscillators.go`, `quant/ta/indicators.go`, tests, `Docs/quant-ta.md`, both CHANGELOGs.
