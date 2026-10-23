# Proposal: ta-ichimoku-chikou-segments

## Why

The owner approved two additions to `ta.Ichimoku` before it merges: the lagging span (Hosoda's close plotted 26 rows behind) and data segments, so a trading halt or a relisting does not leak across the lines the way RSI and KD already refuse to.

## What Changes

- `IchimokuResult` gets `Chikou`: `Chikou[t]` is `close[t+Displacement]`, the last `Displacement` rows of a segment are NaN. The `close` argument is now used and validated for it.
- `IchimokuOptions` gets `SegmentStarts` with the semantics of `RSIOptions.SegmentStarts` and `KDOptions.SegmentStarts`. Every line restarts per segment and the displacement of the cloud and of Chikou never crosses a boundary.
- The zero value of the options gives the same four lines as before.

## Capabilities

### Modified Capabilities

- `ta-trend-indicators`: Ichimoku returns the lagging span and accepts segments.

## Impact

`quant/ta/trend.go`, its tests, `Docs/quant-ta.md`, both CHANGELOGs, the `ta-trend-indicators` spec.
