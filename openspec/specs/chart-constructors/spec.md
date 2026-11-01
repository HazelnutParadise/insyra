# chart-constructors Specification

## Purpose
How the `plot` and `gplot` chart constructors take their data, how they report a chart they cannot build, and how they draw a value that is not a number.

## Requirements

### Requirement: Every chart constructor returns the chart and an error

Every `CreateXxx` function in `plot` and `gplot` SHALL return `(chart, error)`. When it cannot build a chart it SHALL return a nil chart and a non-nil error whose text starts with the package and function name, and it SHALL NOT log anything, not even a warning about an input it skipped on the way to failing. When it builds a chart the error SHALL be nil.

#### Scenario: Nothing to draw
- **WHEN** `plot.CreateBarChart(cfg)` is called with no lists, or `gplot.CreateHistogram(cfg, insyra.NewDataList())` with an empty list
- **THEN** the chart is nil, the error names the function, and nothing is logged

### Requirement: A chart built past a skipped input warns

An outcome that still yields a chart SHALL stay a logged warning with a nil error: a nil list among real ones.

#### Scenario: A nil list among real ones
- **WHEN** `plot.CreateLineChart(cfg, nil, insyra.NewDataList(1, 2, 3))`
- **THEN** a chart comes back with a nil error, and a warning names the dropped position

### Requirement: The gauge takes the same shape

`plot.CreateGaugeChart` SHALL return `(chart, error)` like the other constructors and SHALL document that its error is always nil.

#### Scenario: A gauge
- **WHEN** `plot.CreateGaugeChart(cfg, 42)`
- **THEN** a chart comes back with a nil error

### Requirement: gplot takes insyra's types

`gplot` SHALL take its data as insyra's own types, checked by the compiler:
- `CreateHistogram` takes one `insyra.IDataList`.
- `CreateBarChart`, `CreateLineChart` and `CreateStepChart` take `...insyra.IDataList` and name each series after its list.
- `CreateHeatmapChart` takes an `insyra.IDataTable`, with row *i* of the table as row *i* of the grid.
- `CreateScatterChart` takes `...ScatterSeries`, each with a `Name` and two lists, `X` and `Y`, where point *i* is `(X[i], Y[i])`.

#### Scenario: A plain slice
- **WHEN** a caller passes a `[]float64` to `gplot.CreateBarChart`
- **THEN** the call does not compile; `insyra.NewDataList(values)` does

#### Scenario: Two columns as a scatter series
- **WHEN** `CreateScatterChart(cfg, ScatterSeries{Name: "s", X: dt.GetColByName("h"), Y: dt.GetColByName("w")})` is called with columns of equal length
- **THEN** a chart comes back with one point per row and a nil error

### Requirement: gplot groups bar series

`gplot.CreateBarChart` SHALL draw several lists as grouped bars, side by side in each category, one colour per list, with a legend naming each list.

#### Scenario: Two bar series
- **WHEN** `gplot.CreateBarChart(cfg, a, b)` with two lists of three values
- **THEN** a chart comes back with three categories of two bars each and a legend naming both lists

#### Scenario: Bar series of different lengths
- **WHEN** `gplot.CreateBarChart(cfg, a, b)` where `a` has 3 values and `b` has 2
- **THEN** the chart is nil and the error names list 2 and both lengths

### Requirement: gplot colours every chart from one palette

Every `gplot` chart SHALL colour series *i* with colour *i* of the palette `plot` uses, nine colours from the Insyra website opening with `#2550B0`, `#EE9A2C` and `#D73E3E`. Line and step series SHALL also keep their dash patterns.

#### Scenario: The first series of each chart
- **WHEN** a bar chart, a line chart and a scatter chart are each drawn with two series
- **THEN** the first series of each is drawn in `#2550B0` and the second in `#EE9A2C`

### Requirement: gplot refuses data it cannot pair or draw

A nil list or table given as the only data SHALL be an error. A nil list among real ones in a bar, line or step chart SHALL be dropped with a warning. A nil list SHALL be recognised through `insyra.IsNilList`, so a wrapper type that embeds a nil list counts as nil and does not panic. A scatter series whose `X` or `Y` is nil, or whose `X` and `Y` differ in length, SHALL be an error naming the series.

#### Scenario: X and Y of different lengths
- **WHEN** a `ScatterSeries` has 3 values in `X` and 2 in `Y`
- **THEN** the chart is nil and the error names the series and both lengths

#### Scenario: A nil list inside a wrapper
- **WHEN** `gplot.CreateLineChart(cfg, (*isr.DataList)(nil), list)`
- **THEN** it does not panic, and the chart comes back with one series and a warning naming the dropped position

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

Each chart SHALL draw a cell that is not a Go number as documented in its package page and its doc comment. `gplot`'s bar, histogram, line, step and heat map charts SHALL read cells the way `DataList.ToF64Slice` does: a cell of any Go numeric type is drawn as its value, and any other cell, whether `nil`, a string (including a numeric string such as `"2"`) or a `bool`, is drawn as 0. The scatter charts of both packages SHALL refuse such a cell instead.

#### Scenario: A gplot bar chart over mixed cells
- **WHEN** `gplot.CreateBarChart` is given `NewDataList(1, "2", nil, "abc")`
- **THEN** the bars are 1, 0, 0 and 0

#### Scenario: A gplot heat map over small integer types
- **WHEN** `gplot.CreateHeatmapChart` is given a table whose columns hold `int8`, `uint16` and `float32` values
- **THEN** each cell is drawn as its value, not as 0

### Requirement: How plot draws a value that is not a number

`plot`'s bar and line charts SHALL keep a numeric Y axis only while every cell's text parses as a number. One cell whose text does not parse (`nil` prints as `<nil>`) turns the axis into categories, and every value is then drawn at its category's position. With a numeric axis, a numeric string is drawn as 0. `plot`'s box plot SHALL leave every cell that is not a Go number out of its five-number summary.

#### Scenario: A plot bar chart with one text cell
- **WHEN** `plot.CreateBarChart` is given `NewDataList(1, "abc", 3)`
- **THEN** the Y axis is a category axis labelled `1`, `abc`, `3`

### Requirement: A gplot chart that cannot draw every series returns an error

`gplot`'s bar, line, step and scatter charts SHALL return a nil chart and an error when any series they were given cannot be drawn: a line or step series whose length differs from `XAxis`, a bar series whose length differs from the first, an empty series, or a series holding a NaN or an infinity. The error SHALL name every such series and why, and nothing SHALL be logged for them. A nil list among real ones in a bar, line or step chart SHALL still be dropped with a warning.

#### Scenario: One series of two has the wrong length
- **WHEN** `gplot.CreateLineChart(LineChartConfig{XAxis: []float64{1, 2, 3}}, a, b)` where `a` has 3 values and `b`, named "b", has 2
- **THEN** the chart is nil and the error says series "b" has 2 values but XAxis has 3

#### Scenario: Two series fail for different reasons
- **WHEN** a step chart is given a list of the right length, an empty list named "empty" and a list named "nan" holding a NaN
- **THEN** the chart is nil and the error names both "empty" and "nan", each with its reason

#### Scenario: A scatter series with no points beside a real one
- **WHEN** `CreateScatterChart` is given one series with points and one whose `X` and `Y` are both empty
- **THEN** the chart is nil and the error names the empty series

### Requirement: gplot refuses a setting it cannot honour

A `gplot` constructor SHALL return a nil chart and an error, and SHALL NOT fall back to a default or leave part of the chart out, when a setting cannot be honoured: `CreateStepChart` for a `StepStyle` other than `"pre"`, `"mid"`, `"post"` or empty, and `CreateBarChart` for `ErrorBars` given with more than one list, whose length differs from the data's, or which hold a NaN or an infinity. The error SHALL name the setting and the value or lengths involved, and nothing SHALL be logged.

#### Scenario: A misspelled step style
- **WHEN** `gplot.CreateStepChart(StepChartConfig{StepStyle: "pr"}, list)`
- **THEN** the chart is nil and the error says the StepStyle "pr" is unknown and lists "pre", "mid" and "post"

#### Scenario: Error bars of the wrong length
- **WHEN** `gplot.CreateBarChart(BarChartConfig{ErrorBars: []float64{0.1}}, insyra.NewDataList(1.0, 2.0, 3.0))`
- **THEN** the chart is nil and the error says ErrorBars has 1 values but the data has 3

#### Scenario: An error bar that is NaN
- **WHEN** `ErrorBars` is `{0.1, NaN, 0.3}` for three bars
- **THEN** the chart is nil and the error says the error bars cannot be drawn

#### Scenario: Error bars with two lists
- **WHEN** `gplot.CreateBarChart(BarChartConfig{ErrorBars: []float64{0.1, 0.2}}, a, b)`
- **THEN** the chart is nil and the error says ErrorBars needs exactly one list

### Requirement: plot refuses a value it cannot draw
The system SHALL return a non-nil error naming the value from every `plot` constructor whose data holds a `NaN` or an infinity, and from `CreateBoxPlot` for a list with no numeric value, instead of returning a chart that renders blank.

#### Scenario: NaN in a bar chart
- **WHEN** `plot.CreateBarChart` is given `NewDataList(1.0, math.NaN())`
- **THEN** it returns a nil chart and an error naming the list (counted from one, as the arguments are) and the row index (counted from zero, as Get(i) takes it)

#### Scenario: Infinity in a gauge
- **WHEN** `plot.CreateGaugeChart` is given `math.Inf(1)`
- **THEN** it returns an error

#### Scenario: A box plot list with no number
- **WHEN** `plot.CreateBoxPlot` is given a series holding an all-text list
- **THEN** it returns an error naming the series

#### Scenario: Finite data still draws
- **WHEN** every value is finite
- **THEN** the chart comes back with a nil error

### Requirement: plot and gplot name a shared chart alike

A chart kind both packages draw SHALL have the same function and config names in both: `CreateBarChart`, `CreateLineChart`, `CreateScatterChart` with `ScatterChartConfig`, and `CreateHeatmapChart` with `HeatmapChartConfig`.

#### Scenario: The heat map name
- **WHEN** a caller draws a heat map with either package
- **THEN** the function is `CreateHeatmapChart` and its config is `HeatmapChartConfig`

### Requirement: The former chart names stay one release

`gplot.CreateScatterPlot`, `gplot.ScatterPlotConfig`, `plot.CreateHeatMap`, `plot.HeatMapConfig`, `plot.HeatMapPoint`, `plot.HeatMapAxis`, `plot.NewHeatMapPoint` and `plot.NewHeatMapMissingPoint` SHALL stay for one release as Deprecated aliases or wrappers that behave as the new names do.

#### Scenario: A former name still compiles
- **WHEN** a caller uses `gplot.CreateScatterPlot(gplot.ScatterPlotConfig{}, s)`
- **THEN** it compiles and returns what `gplot.CreateScatterChart(gplot.ScatterChartConfig{}, s)` returns

### Requirement: Both scatter charts take ScatterSeries

`plot.CreateScatterChart` and `gplot.CreateScatterChart` SHALL take `...ScatterSeries`, each with a `Name` and two lists, `X` and `Y`, where point *i* is `(X[i], Y[i])`, and SHALL draw the series in argument order. A series whose `X` or `Y` is nil, whose `X` and `Y` differ in length, or which has no points SHALL be an error naming the series.

#### Scenario: Two columns as a plot scatter series
- **WHEN** `plot.CreateScatterChart(cfg, plot.ScatterSeries{Name: "s", X: x, Y: y})` is called with lists of equal length
- **THEN** a chart comes back with one point per row and a nil error

#### Scenario: Series order
- **WHEN** `plot.CreateScatterChart` is given series named "b" and then "a"
- **THEN** the chart's first series is "b"

### Requirement: A scatter chart refuses a cell that is not a number

Both scatter charts SHALL return a nil chart and an error when a cell of `X` or `Y` is not a Go number, whether `nil`, a string (a numeric string such as `"2"` included) or a `bool`. The error SHALL name the series, the axis and the row index.

#### Scenario: A text cell in X
- **WHEN** a series named "s" has `X` = `NewDataList(1, "two", 3)`
- **THEN** the chart is nil and the error names series "s", X and row index 1

### Requirement: plot refuses an empty list

`plot.CreateBarChart` and `plot.CreateLineChart` SHALL return a nil chart and an error when a list they were given has no values. The check SHALL run before any value is read, so nothing is logged. The error SHALL name the list by its position among the caller's arguments, counted from one with nil arguments included.

#### Scenario: An empty list after a nil one
- **WHEN** `plot.CreateLineChart(cfg, nil, insyra.NewDataList(1, 2), insyra.NewDataList())`
- **THEN** the chart is nil, the error names list 3, and nothing is logged

### Requirement: plot's radar takes its series variadically

`plot.CreateRadarChart` SHALL take `...RadarSeries`, like every other constructor that takes several series.

#### Scenario: Two series
- **WHEN** `plot.CreateRadarChart(cfg, a, b)`
- **THEN** a chart with two series comes back with a nil error

### Requirement: gplot series colours can be set

`gplot`'s bar, line, step and scatter configs SHALL have a `Colors` field that replaces the shared palette, series *i* taking value *i*, repeating when there are more series than values. An empty field SHALL mean the palette.

#### Scenario: Other in gray
- **WHEN** `gplot.CreateBarChart(BarChartConfig{Colors: []string{"#2550B0", "#7F8C8D"}}, a, other)`
- **THEN** the first list is drawn in `#2550B0` and the second in `#7F8C8D`

#### Scenario: A short list repeats
- **WHEN** a line chart has three series and two colours
- **THEN** the third series takes the first colour

### Requirement: gplot refuses a colour it cannot read

A value of `Colors` SHALL be `#RGB`, `#RRGGBB`, `#RRGGBBAA` or a CSS colour name in any case. Any other value SHALL make the constructor return a nil chart and an error naming the position and the value, with nothing logged.

#### Scenario: A misspelled colour
- **WHEN** `gplot.CreateLineChart(LineChartConfig{Colors: []string{"#123456", "nope"}}, a)`
- **THEN** the chart is nil and the error says `Colors[1]` is "nope"
