# chart-constructors Specification

## Purpose
How the `plot` and `gplot` chart constructors take their data, how they report a chart they cannot build, and how they draw a value that is not a number.

## Requirements

### Requirement: Every chart constructor returns the chart and an error

Every `CreateXxx` function in `plot` and `gplot` SHALL return `(chart, error)`. When it cannot build a chart it SHALL return a nil chart and a non-nil error whose text starts with the package and function name, and it SHALL NOT log anything, not even a warning about an input it skipped on the way to failing. When it builds a chart the error SHALL be nil. An outcome that still yields a chart SHALL stay a logged warning with a nil error: a nil list among real ones, a series dropped because its length differs from `XAxis`, error bars of the wrong length. `plot.CreateGaugeChart` SHALL take the same shape and document that its error is always nil.

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

### Requirement: A gplot chart that can draw no series returns an error

`gplot`'s line, step and scatter charts SHALL return an error when none of the series they were given can be drawn, instead of a chart with axes and nothing on it, and that error SHALL name every series and why it could not be drawn. A chart that can draw at least one series SHALL be returned with a nil error, and each series it drops SHALL be named in a warning. A series is not drawn when its length differs from `XAxis`, when it is empty, or when it holds a NaN or an infinity.

#### Scenario: Every series has the wrong length
- **WHEN** `gplot.CreateLineChart(LineChartConfig{XAxis: []float64{1, 2, 3}}, insyra.NewDataList(1, 2))`
- **THEN** the chart is nil and the error says no series could be drawn because series "two" has 2 values but XAxis has 3

#### Scenario: One series of two is drawable
- **WHEN** the same config is given one list of 3 values and one of 2
- **THEN** a chart comes back with a nil error, and a warning names the shorter series

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
