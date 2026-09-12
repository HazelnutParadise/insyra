# Proposal: never-panic-on-bad-chart-input

## Why

`error-philosophy` already says that under the default configuration no insyra package may panic. Eight code paths do, all reached by ordinary bad input rather than by anything exotic. They were found on 2026-09-12 by `test-untested-packages`, the first tests these packages ever had; every one below was reproduced.

| Call | What happens |
| --- | --- |
| `gplot.CreateBarChart(BarChartConfig{}, []float64{1, 2})` | `index out of range [0] with length 0` |
| `gplot.CreateFunctionPlot(FunctionPlotConfig{}, nil)` | nil pointer dereference |
| `gplot.CreateHeatmapChart(cfg, [][]float64{{1,2,3},{4}})` | `index out of range [1] with length 1` |
| `gplot.CreateHeatmapChart(HeatmapChartConfig{Colors: -1}, grid)` | `makeslice: len out of range` |
| `plot.CreateBarChart(cfg, nil)` and the same for line, wordcloud and boxplot | nil pointer dereference |
| `plot.SavePNG(chart, "out")` — a path with no extension | `slice bounds out of range [1:0]`, inside the snapshot dependency |
| `lpgen.ParseLingoModel_str("…\n@BIN)X(;\n…")` | `slice bounds out of range [7:4]` |
| `utils.TruncateString("hello", -1)` | `slice bounds out of range [:-1]` |

The first one is the one that matters most: a zero-value config is the first thing anyone writes, and `XAxis` is the only field on `BarChartConfig` the documentation does not mark optional. The LINGO one is next: that parser's input is a file the user supplies, so malformed input is expected, not exceptional.

`utils.TruncateString` is the exception to "reached by ordinary input": its only callers are in `show.go`, and every width they pass is clamped to at least 20 before it arrives, so no current call can reach the panic. It is fixed anyway because the guard is two lines and the function is an internal helper anyone may call next.

## What Changes

Each one keeps the call usable rather than inventing a new shape:

- **`gplot.CreateBarChart`** skips `NominalX` when `XAxis` is empty. The bars still render, positioned 0..n-1 on a numeric axis, instead of a crash. Auto-generating `"1".."n"` the way `plot` does would also work but invents labels the caller did not ask for; if that is preferred it is a separate decision.
- **`gplot.CreateFunctionPlot`** refuses a nil function, warns and returns nil — the same shape every other constructor in the package uses for input it cannot plot.
- **`gplot.CreateHeatmapChart`** refuses a ragged grid, naming the first row whose length differs, and clamps a non-positive `Colors` to the default the way `0` already is.
- **`plot`'s constructors** skip a nil `IDataList` with a warning, and return nil when that leaves nothing to draw — which is what each already does for no data at all.
- **`plot.SavePNG`** checks the path has an extension and returns an error naming the problem, instead of letting the snapshot dependency slice an empty string.
- **`lpgen`'s LINGO parser** ignores a declaration whose parentheses are out of order, like every other line it cannot read.
- **`utils.TruncateString`** treats a negative width as zero.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `error-philosophy`: the existing "never panics" requirement gains the concrete rule that a constructor given input it cannot use reports it and returns nil, and that a parser given malformed input skips what it cannot read.

## Impact

- `gplot/bar.go`, `gplot/function.go`, `gplot/heatmap.go`; `plot/bar.go`, `plot/line.go`, `plot/wordcloud.go`, `plot/boxplot.go`, `plot/save_chart.go`; `lpgen/lingo.go`; `internal/utils/utils.go`. Tests alongside each.
- User-visible: a call that used to crash the program now returns nil or an error. Both changelogs get entries under `### gplot`, `### plot` and `### lpgen`.
- No signature changes. Nothing that worked before behaves differently — every path changed here previously ended in a panic.
