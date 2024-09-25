# [ gplot ] Package

The `gplot` package is designed to create and save various types of charts using the Gonum library. It provides a simple interface to generate and export charts in different formats such as PNG, PDF, JPEG, and SVG.

## Installation

To install the `gplot` package, use the following command:

```bash
go get github.com/HazelnutParadise/insyra/gplot
```

## Features

### Bar Chart

#### `BarChartConfig`

- `Title`: The title of the chart.
- `XAxis`: Data for the X-axis (categories).
- `SeriesData`: The data for the series. Supported types:
  - `[]float64`
  - `*insyra.DataList`
- `XAxisName`: Optional: Name for the X-axis.
- `YAxisName`: Optional: Name for the Y-axis.
- `BarWidth`: Optional: Width of each bar in the chart. Default is 20.

#### `CreateBarChart(config BarChartConfig) *plot.Plot`

Creates a bar chart based on the provided configuration.

#### Example

![bar_example](./img/gplot_bar_example.png)

### Histogram

#### `HistogramConfig`

- `Title`: The title of the chart.
- `SeriesData`: The data for the histogram. Supported types:
  - `[]float64`
  - `*insyra.DataList`
  - `insyra.IDataList`
- `XAxisName`: Optional: Name for the X-axis.
- `YAxisName`: Optional: Name for the Y-axis.
- `Bins`: Number of bins for the histogram.

#### `CreateHistogram(config HistogramConfig) *plot.Plot`

Creates a histogram based on the provided configuration.

#### Example

![histogram_example](./img/gplot_histogram_example.png)

### Line Chart

#### `LineChartConfig`

- `Title`: The title of the chart.
- `XAxis`: Data for the X-axis (categories).
- `SeriesData`: The data for the series. Supported types:
  - `map[string][]float64`
  - `[]*insyra.DataList`
  - `[]insyra.IDataList`
- `XAxisName`: Optional: Name for the X-axis.
- `YAxisName`: Optional: Name for the Y-axis.

#### `CreateLineChart(config LineChartConfig) *plot.Plot`

Creates a line chart based on the provided configuration.

#### Example

![line_example](./img/gplot_line_example.png)

### Saving Charts

`func SaveChart(plt *plot.Plot, filename string)`

Saves the plot to a file. Supported file formats: `.jpg`, `.jpeg`, `.pdf`, `.png`, `.svg`, `.tex`, `.tif`, `.tiff`
Automatically determine the file extension based on the filename.