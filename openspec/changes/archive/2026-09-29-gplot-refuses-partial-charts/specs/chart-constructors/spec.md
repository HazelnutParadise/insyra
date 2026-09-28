## MODIFIED Requirements

### Requirement: Every chart constructor returns the chart and an error

Every `CreateXxx` function in `plot` and `gplot` SHALL return `(chart, error)`. When it cannot build a chart it SHALL return a nil chart and a non-nil error whose text starts with the package and function name, and it SHALL NOT log anything, not even a warning about an input it skipped on the way to failing. When it builds a chart the error SHALL be nil. An outcome that still yields a chart SHALL stay a logged warning with a nil error: a nil list among real ones, error bars of the wrong length. `plot.CreateGaugeChart` SHALL take the same shape and document that its error is always nil.

#### Scenario: Nothing to draw
- **WHEN** `plot.CreateBarChart(cfg)` is called with no lists, or `gplot.CreateHistogram(cfg, insyra.NewDataList())` with an empty list
- **THEN** the chart is nil, the error names the function, and nothing is logged

#### Scenario: A nil list among real ones
- **WHEN** `plot.CreateLineChart(cfg, nil, insyra.NewDataList(1, 2, 3))`
- **THEN** a chart comes back with a nil error, and a warning names the dropped position

#### Scenario: A gauge
- **WHEN** `plot.CreateGaugeChart(cfg, 42)`
- **THEN** a chart comes back with a nil error

## REMOVED Requirements

### Requirement: A gplot chart that can draw no series returns an error

**Reason**: a chart that could draw some series returned them with a nil error and named the rest only in a warning, so a caller could get fewer series than asked for without noticing.

**Migration**: replaced by "A gplot chart that cannot draw every series returns an error". A call that relied on getting the drawable series back must fix or drop the failing series before calling; the error names each one.

## ADDED Requirements

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
