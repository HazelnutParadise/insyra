## MODIFIED Requirements

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

## ADDED Requirements

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
