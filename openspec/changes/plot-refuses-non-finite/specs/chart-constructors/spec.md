## ADDED Requirements

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
