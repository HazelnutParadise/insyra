## ADDED Requirements

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
