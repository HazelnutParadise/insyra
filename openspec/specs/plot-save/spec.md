# plot-save Specification

## Purpose
Saving a `plot` chart to HTML or PNG either produces the chart or says why it cannot, so that data ECharts' JSON cannot hold never yields a silently empty page.

## Requirements

### Requirement: Saving a chart with unencodable data fails loudly
`plot.SaveHTML` and `plot.SavePNG` SHALL return an error, and SHALL NOT write a file, when the chart's options hold a value JSON cannot encode, such as `NaN` or an infinity.

#### Scenario: A NaN bar
- **WHEN** a bar chart built from a list holding `NaN` is passed to `SaveHTML`
- **THEN** the call returns an error saying the data cannot be encoded
- **AND** no file is written

#### Scenario: Finite data
- **WHEN** a bar chart built from finite values is passed to `SaveHTML`
- **THEN** the file is written and the call returns nil
