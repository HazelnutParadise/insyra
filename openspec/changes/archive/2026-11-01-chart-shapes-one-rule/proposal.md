## Why

`plot` and `gplot` draw the same kinds of chart with different calls (#413): a scatter chart takes `map[string][]ScatterPoint` in `plot` and `...ScatterSeries` in `gplot`, the scatter and heat map functions have different names, `gplot.CreateBarChart` draws one list where `plot`'s draws many, `plot.CreateRadarChart` alone takes a slice, and an empty list is drawn as an empty series by `plot` but refused by `gplot`. Moving a chart from one package to the other means reshaping the data and relearning the rules.

## What Changes

- **BREAKING** `plot.CreateScatterChart` takes `...ScatterSeries` (`Name`, `X`, `Y` lists) like `gplot`; `plot.ScatterPoint` is removed. Series are drawn in argument order.
- **BREAKING** Both scatter charts refuse a cell that is not a Go number, naming the series and row; `gplot` used to draw it as 0.
- `gplot.CreateScatterPlot`/`ScatterPlotConfig` become `CreateScatterChart`/`ScatterChartConfig`; `plot.CreateHeatMap` and its types become `CreateHeatmapChart`, `HeatmapChartConfig`, `HeatmapPoint`, `HeatmapAxis`, `NewHeatmapPoint`, `NewHeatmapMissingPoint`. The old names stay one release as Deprecated wrappers and aliases.
- `gplot.CreateBarChart` takes `...insyra.IDataList` and draws grouped bars; `ErrorBars` is allowed with one list only.
- Every `gplot` chart colours its series from the palette `plot` uses, nine colours from the Insyra website, the last a gray for "Other"; lines and steps, black before, are coloured and keep their dash patterns.
- **BREAKING** `plot.CreateRadarChart` takes `...RadarSeries`.
- **BREAKING** `plot.CreateBarChart` and `CreateLineChart` refuse an empty list, naming its position among the arguments, before reading anything.
- Positions in `plot`'s errors count the caller's arguments, nil ones included.
- `gplot` recognises a nil list wrapped in another type (`isr.DL`) through `insyra.IsNilList`, so it no longer panics on one.

## Impact

- `plot/`, `gplot/`, `internal/dsl/commands/plot.go`, `Docs/plot.md`, `Docs/gplot.md`, `Docs/tutorials/static-executive-report-with-gplot.md`, both changelogs, `AGENTS.md` (follow-up to remove the wrappers), `chart-constructors` spec.
