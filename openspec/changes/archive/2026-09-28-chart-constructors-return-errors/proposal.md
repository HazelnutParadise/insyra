# Proposal: chart-constructors-return-errors

## Why

Finding PL-3 of the API review ([#253](https://github.com/HazelnutParadise/insyra/issues/253)) has three parts.

- The chart constructors have no error. Twelve of the thirteen `plot.CreateXxx` functions and all seven `gplot.CreateXxx` functions return `nil` when they cannot build a chart and log the reason as a warning. AGENTS.md gives an ordinary function one failure shape, `(T, error)`. A caller who does not check for `nil` finds out later: `plot.SaveHTML` panics on a nil chart.
- Six `gplot` constructors take `data any` and sort out its type at run time. A value of the wrong type compiles, logs a warning and returns `nil`. The map forms (`map[string][]float64`, `map[string][][]float64`) assign each series its colour and dash style in map iteration order, which changes from run to run. The list form of `CreateScatterPlot` reads one list as alternating x and y values, a layout no table has.
- The charts draw a value that is not a number in ways nothing documents. The `ToF64Slice` follow-up in AGENTS.md keeps the display paths as they are, but the review asked for the behaviour to be written down. Measured on 2026-09-28, the behaviour differs between the two packages and within `gplot`:
  - `gplot`'s bar, histogram, line, step and scatter charts draw `nil`, `"abc"` and even the numeric string `"2"` as 0.
  - `gplot.CreateHeatmapChart` reads only `float64`, `float32`, `int`, `int32` and `int64`, so an `int8`, `int16` or unsigned column is drawn as all zeros.
  - In `plot`'s bar and line charts, a single cell whose text is not a number (`nil` prints as `<nil>`) turns the whole Y axis into categories and draws every value at its category's position. A numeric string there is drawn as 0.

## What Changes

- **BREAKING**: every chart constructor in `plot` and `gplot` returns `(chart, error)`. A chart it cannot build is a nil chart and an error naming the function and the reason. The warning that used to stand in for the error is gone. Warnings stay only for outcomes that still produce a chart. `plot.CreateGaugeChart` cannot fail today and returns a nil error. It takes the same shape so the family has one signature pattern, and so a later check on its value does not break it again.
- **BREAKING**: `gplot` takes insyra's types instead of `any`:
  - `CreateBarChart(config, data insyra.IDataList)` and `CreateHistogram(config, data insyra.IDataList)`
  - `CreateLineChart(config, data ...insyra.IDataList)` and `CreateStepChart(config, data ...insyra.IDataList)`, the shape `plot.CreateLineChart` already has
  - `CreateHeatmapChart(config, data insyra.IDataTable)`
  - `CreateScatterPlot(config, series ...ScatterSeries)` with a new `ScatterSeries{Name string; X, Y insyra.IDataList}`

  The slice and map forms go away. A plain slice is passed as `insyra.NewDataList(values)` and a grid as `insyra.ReadSlice2D(grid)`. A nil list among real ones in a line or step chart is dropped with a warning, the way `plot` drops one. `X` and `Y` of different lengths are an error.
- A `gplot` line, step or scatter chart that cannot draw any of its series returns an error. It used to return a chart with nothing on it.
- A failing constructor logs nothing: the warnings for a skipped nil list or series are logged only when a chart comes back.
- `gplot` refuses a value its charts cannot draw, each an error instead of a panic, a hang or a wrong chart, found by the review of this change and present before it: `CreateHistogram` refuses a `NaN` or an infinity, `CreateHeatmapChart` an infinity or a table of `NaN` alone, and `CreateFunctionPlot` a bound that is not finite. The error for a line, step or scatter chart that can draw no series names every series and why.
- `gplot.CreateHeatmapChart` reads each cell the way `ToF64Slice` does, so every numeric type is drawn as its value and every `gplot` chart follows one rule for a value that is not a number.
- `plot.CreateBoxPlot` names each series it drops for having no lists, and its error says that no series has data, instead of "no series provided" when a series was provided.
- How each chart draws a value that is not a number is written in `Docs/plot.md`, `Docs/gplot.md`, `Docs/cli-dsl.md` and each constructor's Go doc comment. Nothing about that behaviour changes except the heat map's numeric types.
- The CLI's `plot` command reports the constructor's error.

## Capabilities

### New Capabilities

- `chart-constructors`: how the chart constructors report a chart they cannot build, the data types `gplot` takes, and how a value that is not a number is drawn.

### Modified Capabilities

- `error-philosophy`: "A constructor given input it cannot use reports it and returns nil" becomes "A constructor given input it cannot use returns an error".
- `test-suite-integrity`: "A chart is tested by rendering it" asks for a test of each documented error rather than of each documented `nil`.

## Impact

- `plot/*.go` (thirteen constructors), `gplot/*.go` (seven constructors, the new `ScatterSeries`, the helpers that duplicated `*DataList` and `IDataList` paths), `cli/commands/plot.go`.
- Tests: `plot/charts_test.go`, `plot/no_panic_test.go`, `plot/heatmap_point_test.go`, `gplot/charts_test.go`, `gplot/no_panic_test.go`, `gplot/step_test.go`, `gplot/save_chart_test.go`, and new tests for the error shape, the typed data, the documented values and the CLI's `plot` error.
- `Docs/plot.md`, `Docs/gplot.md`, `Docs/cli-dsl.md`, the three chart tutorials under `Docs/tutorials/`, `skills/insyra/SKILL.md` (the error-shape principle now covers the chart constructors), both changelogs, `api-review.md` (PL-3), `delivery-status.md`, and the `AGENTS.md` follow-ups that describe the old `nil` returns.
- Not in this change, recorded as `AGENTS.md` follow-ups:
  - A NaN or ±Inf in a `plot` chart, or a box plot list with no number in it, gives a page with no chart and no error, because go-echarts drops its encoding error.
  - `SaveHTML`/`SavePNG` still panic on a nil chart a caller passes after ignoring the error.
  - `insyra.ReadSlice2D`, which the docs now point `[][]float64` callers at, drops what a row longer than the first holds.
  - `plot.CreateWordCloud` panics on a slice cell, and both packages panic on an `isr` wrapper holding a nil list.
