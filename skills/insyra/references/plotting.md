# plot and gplot

Two packages, two purposes:

- **`plot`** builds interactive charts with go-echarts and saves them as HTML.
  Use it when the chart is going to be looked at in a browser.
- **`gplot`** builds static charts with gonum/plot and saves them as an image.
  Use it when the chart is going into a document, or when no browser is
  involved.

Neither returns an error from the constructor: `CreateXxx` returns `nil` when it
cannot build the chart and records the reason through the logger. Check for nil
before saving.

## plot

```go
chart := plot.CreateBarChart(plot.BarChartConfig{
    Title: "Revenue",
    XAxis: []string{"Q1", "Q2", "Q3", "Q4"},
}, revenueList)                       // one or more insyra.IDataList
if chart == nil {
    // no data, or every list was nil
}
err := plot.SaveHTML(chart, "revenue.html")
```

| Constructor | Data argument |
| --- | --- |
| `CreateBarChart(cfg, data ...IDataList)` | one list per series |
| `CreateLineChart(cfg, data ...IDataList)` | one list per series |
| `CreateScatterChart(cfg, map[string][]ScatterPoint)` | named series of `{X, Y}` |
| `CreatePieChart(cfg, items ...PieItem)` | `{Name, Value}` per slice |
| `CreateBoxPlot(cfg, series ...BoxPlotSeries)` | each series holds `Data []IDataList`, one list per category |
| `CreateHeatMap(cfg, points ...)` | built with `HeatMapPoint(x, y, v)` / `HeatMapMissingPoint(x, y)` |
| `CreateRadarChart(cfg, []RadarSeries)` | `Values []float32`, one per indicator |
| `CreateKlineChart(cfg, points ...KlinePoint)` | `{Date, Open, High, Low, Close}` |
| `CreateFunnelChart(cfg, map[string]float64)` | stage → value |
| `CreateGaugeChart(cfg, value float64)` | one number; never returns nil |
| `CreateSankeyChart(cfg, links ...SankeyLink)` | `{Source, Target, Value}` |
| `CreateThemeRiverChart(cfg, data ...ThemeRiverData)` | `{Date, Name, Value}` |
| `CreateWordCloud(cfg, data IDataList)` | one list |

Every config starts with the same block — `Width`, `Height`,
`BackgroundColor`, `Theme`, `Title`, `Subtitle`, `TitlePos` — and all of it is
optional. `Title` and `Subtitle` are HTML-escaped, so user data cannot inject
markup into the page.

### Saving

- `SaveHTML(chart, path, animation ...bool)` writes a self-contained page. It
  needs nothing but a writable path. Pass `false` to turn the animation off.
- `SavePNG(chart, path, useOnlineServiceOnFail ...bool)` **needs a local Chrome
  or Chromium**. The path must carry an extension, which is what picks the image
  format. When the local render fails and no third argument is given, it falls
  back to an online rendering service by default, which uploads the chart and
  all of its data. Pass `false` to keep the data local: the call then returns
  the render error instead.

In a script or on a server, prefer `SaveHTML`.

## gplot

```go
plt := gplot.CreateLineChart(gplot.LineChartConfig{
    Title: "Daily close",
}, map[string][]float64{"close": closes})
if plt == nil {
    // unsupported data type
}
gplot.SaveChart(plt, "close.png")   // .jpg .jpeg .pdf .png .svg .tex .tif .tiff
```

| Constructor | Data argument |
| --- | --- |
| `CreateBarChart(cfg, data any)` | `[]float64`, `*DataList` or `IDataList` |
| `CreateLineChart(cfg, data any)` | `map[string][]float64`, `[]*DataList` or `[]IDataList` |
| `CreateStepChart(cfg, data any)` | as line; `StepStyle` is `"pre"`, `"mid"` or `"post"` |
| `CreateScatterPlot(cfg, data any)` | `map[string][][]float64` of `[x, y]` pairs, or lists read as alternating x and y |
| `CreateHistogram(cfg, data any)` | `[]float64`, `*DataList` or `IDataList` |
| `CreateFunctionPlot(cfg, func(float64) float64)` | the function to draw |
| `CreateHeatmapChart(cfg, data any)` | `[][]float64`, `*DataTable` or `IDataTable`; no row may be shorter than row 0, and values past row 0's length are ignored |

`SaveChart` returns nothing. A bad path or an unsupported extension goes to
`insyra.LogFatal`, which ends the program unless
`insyra.Config.SetDontPanic(true)` is set, in which case it is only logged.

`CreateBarChart` with no `XAxis` numbers the bars 1, 2, 3, … `CreateHistogram`
with `Bins` at or below zero uses 10. `CreateFunctionPlot` samples 100 points
per unit of x, so a range of ±1e6 is effectively a hang, not an error.

## Things that are easy to get wrong

- **A `nil` list is skipped, not fatal.** `plot`'s bar, line and box charts drop
  a nil entry with a warning and draw the rest, returning nil only when nothing
  is left. It is still worth not passing one.
- **Mismatched lengths are silent.** `gplot`'s line and step charts drop a
  series whose length does not match `XAxis` and still return the chart.
- **Some constructors mutate what you pass.** `CreateKlineChart` sorts its
  points in place; `CreateBoxPlot` and `CreateRadarChart` fill in each series'
  `Color`; `CreateRadarChart` also writes into `config.MaxValues`. Do not reuse
  one fixture slice across charts and expect it unchanged.
- **The rendered HTML is not reproducible.** go-echarts assigns a random chart
  id, so compare on substrings, never on a whole file.
- `CreateFunnelChart` and `CreateWordCloud` iterate a map, so series order
  varies between runs.
