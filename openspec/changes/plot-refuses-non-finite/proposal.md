# Proposal: plot-refuses-non-finite

## Why

[#409](https://github.com/HazelnutParadise/insyra/issues/409): go-echarts writes a chart's options with `encoding/json`, which refuses `NaN` and an infinity, and drops the error. One such value left `SaveHTML` writing a page with no chart and no error. A box plot list that is empty or all text has a `NaN` five-number summary and did the same from valid-looking input.

## What Changes

- Every `plot` constructor returns an error naming the value when its data holds a `NaN` or an infinity (`CreateBarChart`, `CreateLineChart`, `CreateBoxPlot`, `CreateFunnelChart`, `CreateGaugeChart`, `CreateKlineChart`, `CreatePieChart`, `CreateRadarChart`, `CreateSankeyChart`, `CreateScatterChart`, `CreateThemeRiverChart`).
- `CreateBoxPlot` also refuses a list with no number in it.
- Chosen from the issue's two options: refuse at construction, as `gplot` does, not draw a gap. It is simpler and makes the problem visible; `CreateHeatMap` keeps writing a missing point as `"-"`.

## Capabilities

### Modified Capabilities

- `chart-constructors`: a value a chart cannot draw is an error.

## Impact

`plot/*.go`, `plot/finite.go`, `plot/finite_test.go`, `Docs/plot.md`, `AGENTS.md` (the follow-up is resolved and removed), both CHANGELOGs. Breaking: a call that returned a blank-page chart now returns an error, on the `0.4` line only.
