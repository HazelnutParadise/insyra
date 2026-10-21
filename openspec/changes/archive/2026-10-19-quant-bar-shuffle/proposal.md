# Proposal: quant-bar-shuffle

## Why

[#471](https://github.com/HazelnutParadise/insyra/issues/471): a pattern that appears as often in shuffled prices as in real ones is an artefact of how it is drawn. The study needed a shuffled-bar random walk as the null.

## What Changes

- `quant.ShuffleBars(open, high, low, close, opts)` returns a table of bars with the days in random order: each bar keeps its gap from the previous close, its high, low and close over its open, and its volume, and the path is rebuilt from the first bar.
- `ShuffleOptions.Seed` always applies, and `SegmentStarts` shuffles within each segment.

## Capabilities

### New Capabilities

- `quant-bar-shuffle`: a random-walk copy of a series of daily bars made by shuffling whole bars, the null against which price patterns are compared.

## Impact

`quant/`, tests, `Docs/quant.md`, both CHANGELOGs.
