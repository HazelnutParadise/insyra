# Proposal: plot-save-refuses-non-finite

## Why

[#409](https://github.com/HazelnutParadise/insyra/issues/409): go-echarts writes chart options with `encoding/json`, which rejects `NaN` and infinities, and drops that error. A chart with one such value was saved as a page with no chart and no error. [#412](https://github.com/HazelnutParadise/insyra/issues/412) also left `gplot.CreateBarChart` and `CreateHistogram` panicking on a nil list wrapper such as `isr.DL`.

## What Changes

- `plot.SaveHTML` and `plot.SavePNG` encode the chart's options first and return an error naming the problem when they cannot be encoded.
- `gplot.CreateBarChart` and `CreateHistogram` treat a nil list wrapper like a nil list: a warning and a nil result.
- Not changed: the `plot` constructors still return the chart. Returning an error from them, or drawing the value as a gap, changes every caller and belongs to 0.4.

## Capabilities

### New Capabilities

- `plot-save`: saving a `plot` chart never produces a silently empty page.

## Impact

`plot/save_chart.go`, `gplot/bar.go`, `gplot/histogram.go`, tests, `Docs/plot.md`, both CHANGELOGs.
