# chart-constructors Specification

## Purpose
How the `plot` and `gplot` chart constructors take their data, how they report a chart they cannot build, and how they draw a value that is not a number.

## Requirements

### Requirement: Every chart constructor returns the chart and an error

Every `CreateXxx` function in `plot` and `gplot` SHALL return `(chart, error)`. When it cannot build a chart it SHALL return a nil chart and a non-nil error whose text starts with the package and function name, and it SHALL NOT log anything, not even a warning about an input it skipped on the way to failing. When it builds a chart the error SHALL be nil. An outcome that still yields a chart SHALL stay a logged warning with a nil error: a nil list among real ones. `plot.CreateGaugeChart` SHALL take the same shape and document that its error is always nil.

#### Scenario: Nothing to draw
- **WHEN** `plot.CreateBarChart(cfg)` is called with no lists, or `gplot.CreateHistogram(cfg, insyra.NewDataList())` with an empty list
- **THEN** the chart is nil, the error names the function, and nothing is logged

#### Scenario: A nil list among real ones
- **WHEN** `plot.CreateLineChart(cfg, nil, insyra.NewDataList(1, 2, 3))`
- **THEN** a chart comes back with a nil error, and a warning names the dropped position

#### Scenario: A gauge
- **WHEN** `plot.CreateGaugeChart(cfg, 42)`
- **THEN** a chart comes back with a nil error

### Requirement: gplot takes insyra's types

`gplot` SHALL take its data as insyra's own types, checked by the compiler:
- `CreateBarChart` and `CreateHistogram` take one `insyra.IDataList`.
- `CreateLineChart` and `CreateStepChart` take `...insyra.IDataList` and name each series after its list.
- `CreateHeatmapChart` takes an `insyra.IDataTable`, with row *i* of the table as row *i* of the grid.
- `CreateScatterPlot` takes `...ScatterSeries`, each with a `Name` and two lists, `X` and `Y`, where point *i* is `(X[i], Y[i])`.

A nil list or table given as the only data SHALL be an error. A nil list among real ones in a line or step chart SHALL be dropped with a warning. A scatter series whose `X` or `Y` is nil, or whose `X` and `Y` differ in length, SHALL be an error naming the series.

#### Scenario: A plain slice
- **WHEN** a caller passes a `[]float64` to `gplot.CreateBarChart`
- **THEN** the call does not compile; `insyra.NewDataList(values)` does

#### Scenario: Two columns as a scatter series
- **WHEN** `CreateScatterPlot(cfg, ScatterSeries{Name: "s", X: dt.GetColByName("h"), Y: dt.GetColByName("w")})` is called with columns of equal length
- **THEN** a chart comes back with one point per row and a nil error

#### Scenario: X and Y of different lengths
- **WHEN** a `ScatterSeries` has 3 values in `X` and 2 in `Y`
- **THEN** the chart is nil and the error names the series and both lengths

### Requirement: gplot refuses a value it cannot draw

A `gplot` constructor SHALL return a nil chart and an error, and SHALL NOT panic or hang, when a value would leave it unable to draw or save the chart: `CreateHistogram` for a NaN or an infinity in its data, `CreateHeatmapChart` for an infinity in its table or a table of NaN alone, and `CreateFunctionPlot` for an `XMin`, `XMax`, `YMin` or `YMax` that is a NaN or an infinity. The error SHALL say which value. A NaN among numbers in a heat map SHALL be drawn as an empty cell.

#### Scenario: A histogram with NaN
- **WHEN** `CreateHistogram(cfg, insyra.NewDataList(1.0, math.NaN(), 3.0))` on amd64 or arm64
- **THEN** the chart is nil and the error names index 1

#### Scenario: A heat map with an infinity
- **WHEN** `CreateHeatmapChart` is given a table whose cell at row 1, column 0 is `+Inf`
- **THEN** the chart is nil and the error names that cell

#### Scenario: An infinite function range
- **WHEN** `CreateFunctionPlot(FunctionPlotConfig{XMax: math.Inf(1)}, math.Sin)`
- **THEN** it returns at once with a nil chart and an error naming `XMax`

### Requirement: How a value that is not a number is drawn

Each chart SHALL draw a cell that is not a Go number as documented in its package page and its doc comment, and this change SHALL NOT alter that behaviour except for the `gplot` heat map's numeric types:
- `gplot`'s bar, histogram, line, step, scatter and heat map charts SHALL read cells the way `DataList.ToF64Slice` does: a cell of any Go numeric type is drawn as its value, and any other cell, whether `nil`, a string (including a numeric string such as `"2"`) or a `bool`, is drawn as 0.
- `plot`'s bar and line charts SHALL keep a numeric Y axis only while every cell's text parses as a number. One cell whose text does not parse (`nil` prints as `<nil>`) turns the axis into categories, and every value is then drawn at its category's position. With a numeric axis, a numeric string is drawn as 0.
- `plot`'s box plot SHALL leave every cell that is not a Go number out of its five-number summary.

#### Scenario: A gplot bar chart over mixed cells
- **WHEN** `gplot.CreateBarChart` is given `NewDataList(1, "2", nil, "abc")`
- **THEN** the bars are 1, 0, 0 and 0

#### Scenario: A gplot heat map over small integer types
- **WHEN** `gplot.CreateHeatmapChart` is given a table whose columns hold `int8`, `uint16` and `float32` values
- **THEN** each cell is drawn as its value, not as 0

#### Scenario: A plot bar chart with one text cell
- **WHEN** `plot.CreateBarChart` is given `NewDataList(1, "abc", 3)`
- **THEN** the Y axis is a category axis labelled `1`, `abc`, `3`

### Requirement: A gplot chart that cannot draw every series returns an error

`gplot`'s line, step and scatter charts SHALL return a nil chart and an error when any series they were given cannot be drawn: a line or step series whose length differs from `XAxis`, an empty series, or a series holding a NaN or an infinity. The error SHALL name every such series and why, and nothing SHALL be logged for them. A nil list among real ones in a line or step chart SHALL still be dropped with a warning.

#### Scenario: One series of two has the wrong length
- **WHEN** `gplot.CreateLineChart(LineChartConfig{XAxis: []float64{1, 2, 3}}, a, b)` where `a` has 3 values and `b`, named "b", has 2
- **THEN** the chart is nil and the error says series "b" has 2 values but XAxis has 3

#### Scenario: Two series fail for different reasons
- **WHEN** a step chart is given a list of the right length, an empty list named "empty" and a list named "nan" holding a NaN
- **THEN** the chart is nil and the error names both "empty" and "nan", each with its reason

#### Scenario: A scatter series with no points beside a real one
- **WHEN** `CreateScatterPlot` is given one series with points and one whose `X` and `Y` are both empty
- **THEN** the chart is nil and the error names the empty series

### Requirement: gplot refuses a setting it cannot honour

A `gplot` constructor SHALL return a nil chart and an error, and SHALL NOT fall back to a default or leave part of the chart out, when a setting cannot be honoured: `CreateStepChart` for a `StepStyle` other than `"pre"`, `"mid"`, `"post"` or empty, and `CreateBarChart` for `ErrorBars` whose length differs from the data's or which hold a NaN or an infinity. The error SHALL name the setting and the value or lengths involved, and nothing SHALL be logged.

#### Scenario: A misspelled step style
- **WHEN** `gplot.CreateStepChart(StepChartConfig{StepStyle: "pr"}, list)`
- **THEN** the chart is nil and the error says the StepStyle "pr" is unknown and lists "pre", "mid" and "post"

#### Scenario: Error bars of the wrong length
- **WHEN** `gplot.CreateBarChart(BarChartConfig{ErrorBars: []float64{0.1}}, insyra.NewDataList(1.0, 2.0, 3.0))`
- **THEN** the chart is nil and the error says ErrorBars has 1 values but the data has 3

#### Scenario: An error bar that is NaN
- **WHEN** `ErrorBars` is `{0.1, NaN, 0.3}` for three bars
- **THEN** the chart is nil and the error says the error bars cannot be drawn
