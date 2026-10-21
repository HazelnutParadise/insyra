# Proposal: kd-segment-starts

## Why

[#475 技術指標 SMA/EMA/KD](https://github.com/HazelnutParadise/insyra/issues/475) shipped `ta.KD` without a way to say where a stock's data breaks (a trading halt, a relisting, a price-basis change). KD then carries K, D and its high-low window across the break, so rows after it mix two histories. The other `ta` tools that take `SegmentStarts` restart there; KD should too.

## What Changes

- `KDOptions` gains `SegmentStarts []int`, the rows on which a new data segment starts. Each segment is computed on its own for both `KDRecursive` and `KDMovingAverage`: K and D restart from their initial value, the rolling highest-high and lowest-low windows restart, and a segment shows NaN through its own warm-up.
- A start outside the series is an error. The zero value changes nothing.
- The helpers `cleanSegmentStarts` and `segmentBounds` are added to `quant/ta/zigzag.go` exactly as the volatility-zigzag branch defines them, so the two merge without a second copy.

## Capabilities

### Modified Capabilities

- `quant-technical-indicators`: KD restarts at each segment start.

## Impact

`quant/ta/indicators.go`, `quant/ta/zigzag.go` (helpers), tests, `Docs/quant-ta.md`, both CHANGELOGs.
