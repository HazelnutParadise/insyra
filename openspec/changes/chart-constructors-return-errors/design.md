# Design: chart-constructors-return-errors

## Decisions

### Every constructor returns `(chart, error)`, the gauge included

A caller writes `chart, err := plot.CreateXxx(...)` for every chart and never has to remember which one is different. `CreateGaugeChart` has no failure path today, and its error is documented as always nil, the way `strings.Builder.WriteString` documents its own. `0.4` is the breaking line. Giving the gauge an error now costs nothing, while adding one later, for example to refuse a `NaN` value that currently leaves the page blank, would break its signature a second time.

### An error replaces the warning; warnings stay for partial outcomes

A constructor that cannot build a chart returns the error and does not also log it. That matches the rest of the `(T, error)` functions, `stats` for one: the caller decides whether to log. It also keeps `Config.SetPanicOnError(true)` from panicking on a failure the caller is about to handle. Some outcomes still produce a chart and stay warnings with a nil error:
- a nil list among real ones (plot bar, line, box plot; gplot line, step)
- a series whose length differs from `XAxis` (gplot line, step)
- `ErrorBars` of the wrong length (gplot bar)
- an unknown `StepStyle`, which falls back to `"post"`
- `MaxValues` keys missing from `Indicators` (plot radar)

These were documented as warnings before, and changing them is not part of PL-3.

The one warning that becomes an error is `gplot`'s line, step or scatter chart drawing nothing at all. Once the constructor has an error to return, handing back an empty chart that saves without complaint is the failure the finding describes. The error names every series and why, because a caller who passes the error on, to an HTTP response or a CLI user, has no log to read. `plot` refuses a call left with no list at all, but still draws an empty list as an empty series; changing that is not part of PL-3.

A failing call logs nothing. The warnings for a skipped nil list or series are held until the constructor knows a chart comes back, and an empty list is measured with `Len()` instead of being read through `ToF64Slice`, which logs its own warning for one.

Error text follows `gplot.SaveChart`: `"<package>: <Function>: <reason>"`.

### `gplot` takes `IDataList` and `IDataTable`

AGENTS.md: "Sub-packages take the interfaces as parameter types." `plot` already takes `...insyra.IDataList` for its bar and line charts. The slice and map forms had no capability the list forms lack. A slice is `insyra.NewDataList(values)` (the constructor spreads a slice into cells) and a grid is `insyra.ReadSlice2D(grid)`. Keeping both forms would mean two functions per chart, which the one-name ruling on #211 rules out, or keeping `any`.

Line and step take a variadic `...insyra.IDataList` rather than a slice, the same as `plot.CreateLineChart`. A caller with a slice writes `lists...`.

### Scatter takes `ScatterSeries{Name, X, Y}`

The old list form read one list as `x0, y0, x1, y1, ...`. No table stores points that way, and every plotting library takes x and y separately: gonum's `XYs`, matplotlib's `scatter(x, y)`, R's `plot(x, y)`. Two columns of a table map straight onto `X` and `Y`. `X` and `Y` of different lengths, or a nil `X` or `Y`, are an error. The old code either dropped the odd last value or left a malformed map pair as a point at (0, 0), and there is no right way to pair up two lists of different lengths. Every old call stops compiling, so no call changes meaning silently.

### The heat map reads cells the way `ToF64Slice` does

`convertDataTableToGrid` had its own type switch that knew five numeric types and drew everything else as 0, so an `int8` or `uint` column came out as zeros while the same column in a bar chart was drawn correctly. Reading each column through `ToF64Slice` gives every `gplot` chart one rule: a cell that is a Go number is drawn as its value, and any other cell is drawn as 0. That rule is the `ToF64Slice` follow-up, left as it is. `NewDataTable` pads a shorter column with `nil`, so an uneven table is still a rectangular grid, with 0 in the padding as before. Since a table cannot be ragged, the ragged-grid check goes away with the `[][]float64` form.

### gplot refuses a value its charts cannot draw

The adversarial review measured three constructors that returned a chart they could not draw, all present before this change. `CreateHistogram` with a `NaN` panicked inside gonum's binning on amd64 and drew a wrong chart on arm64. `CreateHeatmapChart` with an infinity, or with nothing but `NaN`, returned a chart whose save panicked. `CreateFunctionPlot` with an infinite X range, or a `NaN` bound, hung. Each now returns an error, the shape the bar chart already had for `NaN` through gonum's own check. A `NaN` among numbers in a heat map stays allowed, because gonum draws it as an empty cell.

### Documenting the non-numeric behaviour without changing it

The follow-up keeps the display paths as they are, so this change describes each chart's behaviour where a caller reads it: the package docs and each constructor's doc comment. `plot`'s bar and line charts do not substitute 0 for text. Any cell whose text does not parse as a number switches the Y axis to categories. Their docs say so instead of repeating the `gplot` rule.

## Risks

- Every chart call in user code stops compiling. The fix is mechanical (`chart, err :=`), and for `gplot` the data has to be wrapped once. The changelog shows each old and new form.
- The `gplot` line, step and scatter charts now return an error where they used to return a chart that saved as an empty plot. A caller who relied on getting an empty chart gets an error instead.
