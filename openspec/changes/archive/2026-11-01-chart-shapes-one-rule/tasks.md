## 1. Scatter
- [x] 1.1 `plot.ScatterSeries`, `plot.CreateScatterChart(config, ...ScatterSeries)`, remove `ScatterPoint`, argument order
- [x] 1.2 Strict numeric reading for both scatter charts, error names series and row
- [x] 1.3 `gplot.CreateScatterChart`/`ScatterChartConfig`, Deprecated `CreateScatterPlot`/`ScatterPlotConfig`

## 2. Names
- [x] 2.1 `plot.CreateHeatmapChart` and its types, Deprecated old names

## 3. gplot bar
- [x] 3.1 `CreateBarChart(config, ...insyra.IDataList)` with grouped bars and a legend
- [x] 3.2 `ErrorBars` refused with more than one list
- [x] 3.3 One palette for every gplot chart; lines keep their dash patterns

## 4. plot rules
- [x] 4.1 `CreateRadarChart(config, ...RadarSeries)`
- [x] 4.2 Empty list refused in bar and line, before any read
- [x] 4.3 Positions in errors count the caller's arguments
- [x] 4.4 `gplot` uses `insyra.IsNilList`/`IsNilTable`

## 5. Callers and docs
- [x] 5.1 CLI `plot` command
- [x] 5.2 `Docs/plot.md`, `Docs/gplot.md`, tutorial
- [x] 5.3 CHANGELOG.md, CHANGELOG_TW.md
- [x] 5.4 AGENTS.md follow-up to remove the Deprecated names

## 6. Verify
- [x] 6.1 Tests for every item, `go test ./plot/... ./gplot/... ./internal/dsl/...`, golangci-lint
- [x] 6.2 Render each changed chart and look at it
