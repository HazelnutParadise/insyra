# Proposal: ta-candle-segments-and-shadow-ratios

## Why

`ta.CandlePatterns` looks back at earlier bars (engulfing, harami, stars, soldiers and crows, and the trend before a hammer), so across a trading halt or a relisting a pattern mixes two histories, which the other `ta` indicators already avoid through `SegmentStarts`. And `CandleOptions.ShadowRatio` served two rules at once (how long the long shadow must be, and how small the opposite shadow must be), so one value could not tune either.

## What Changes

- `CandleOptions` gains `SegmentStarts []int`: each data segment is computed on its own, so no pattern uses a bar of an earlier segment and the first rows of a segment that lack the history a pattern needs are 0. The helpers `cleanSegmentStarts` and `segmentBounds` are reused; an empty list changes nothing.
- `ShadowRatio` is split into `LongShadowRatio` (the long shadow is at least the body times it) and `ShortShadowRatio` (the opposite shadow is at most the long shadow divided by it; a soldier's or crow's shadow is at most the body divided by it). Both default to 2, so the default options give the same flags as before. `ShadowRatio` is removed; it was never released.

## Capabilities

### Modified Capabilities

- `ta-candlestick-patterns`: the cut-offs gain two shadow ratios in place of one, and segments restart the patterns.

## Impact

`quant/ta/candles.go`, `quant/ta/candles_segment_test.go`, `Docs/quant-ta.md`, both CHANGELOGs.
