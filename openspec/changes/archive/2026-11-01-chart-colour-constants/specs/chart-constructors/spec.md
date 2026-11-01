## ADDED Requirements

### Requirement: The palette has named colours

`plot` and `gplot` SHALL export the palette as string constants in palette order, `ColorBlue`, `ColorOrange`, `ColorRed`, `ColorTeal`, `ColorPurple`, `ColorPink`, `ColorGreen`, `ColorBrightBlue` and `ColorGray`, and the brand's background colour as `ColorCream`. Series *i* drawn without `Colors` SHALL use the constant at position *i*.

#### Scenario: Other in gray by name
- **WHEN** `gplot.CreateBarChart(BarChartConfig{Colors: []string{gplot.ColorBlue, gplot.ColorGray}}, a, other)`
- **THEN** the second list is drawn in the palette's gray

### Requirement: gplot chart backgrounds can be set

Every `gplot` config SHALL have a `BackgroundColor` field, read like a value of `Colors`. An empty field SHALL keep white. An unreadable value SHALL make the constructor return a nil chart and an error naming the value, with nothing logged.

#### Scenario: A cream background
- **WHEN** `gplot.CreateLineChart(LineChartConfig{BackgroundColor: gplot.ColorCream}, a)`
- **THEN** the saved chart has the background `#FFF4E6`

#### Scenario: A misspelled background
- **WHEN** `gplot.CreateHistogram(HistogramConfig{BackgroundColor: "nope"}, a)`
- **THEN** the chart is nil and the error says `BackgroundColor` is "nope"
