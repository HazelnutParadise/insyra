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
- `Data`: The data for the series. Supported types:
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
- `Data`: The data for the histogram. Supported types:
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
- `Data`: The data for the series. Supported types:
  - `map[string][]float64`
  - `[]*insyra.DataList`
  - `[]insyra.IDataList`
- `XAxisName`: Optional: Name for the X-axis.
- `YAxisName`: Optional: Name for the Y-axis.

#### `CreateLineChart(config LineChartConfig) *plot.Plot`

Creates a line chart based on the provided configuration.

#### Example

![line_example](./img/gplot_line_example.png)

### Scatter Plot

#### `ScatterPlotConfig`

- `Title`: The title of the chart.
- `Data`: The data for the scatter plot. Supported types:
  - `map[string][][]float64`: A map where keys are series names, and values are two-dimensional data (X, Y pairs).
  - `[]*insyra.DataList`: A slice of DataList pointers, where each DataList contains alternating X and Y values.
  - `[]insyra.IDataList`: A slice of IDataList interfaces, where each contains alternating X and Y values.
- `XAxisName`: Optional: Name for the X-axis.
- `YAxisName`: Optional: Name for the Y-axis.

#### `CreateScatterPlot(config ScatterPlotConfig) *plot.Plot`

Creates a scatter plot based on the provided configuration. Each series is displayed with different colors and shapes to distinguish them.

#### Example

```go
package main

import (
	"github.com/HazelnutParadise/insyra/gplot"
)

func main() {
	// Create scatter plot data
	data := map[string][][]float64{
		"Series A": {
			{1.0, 2.0},
			{2.0, 4.0},
			{3.0, 6.0},
			{4.0, 8.0},
			{5.0, 10.0},
		},
		"Series B": {
			{1.0, 1.0},
			{2.0, 3.0},
			{3.0, 5.0},
			{4.0, 7.0},
			{5.0, 9.0},
		},
	}

	config := gplot.ScatterPlotConfig{
		Title:     "Sample Scatter Plot",
		Data:      data,
		XAxisName: "X Axis",
		YAxisName: "Y Axis",
	}

	plt := gplot.CreateScatterPlot(config)
	gplot.SaveChart(plt, "scatter_plot.png")
}
```

### Function Plot

#### `FunctionPlotConfig`

- `Title`: The title of the chart.
- `XAxis`: Label for the X-axis.
- `YAxis`: Label for the Y-axis.
- `Func`: The mathematical function to plot.
- `XMin`: Minimum value of X (optional).
- `XMax`: Maximum value of X (optional).
- `YMin`: Minimum value of Y (optional).
- `YMax`: Maximum value of Y (optional).

#### `CreateFunctionPlot(config FunctionPlotConfig) *plot.Plot`

Creates a function plot based on the provided configuration.

#### Usage of `FunctionPlotConfig.Func`

The `Func` field is a function that takes a float64 value as input and returns a float64 value as output. This function is used to generate the data points for the plot.

```go
func(x float64) float64 {
 return x * x
}
```

#### Example

![function_example](./img/gplot_function_example.png)

### Heatmap

#### `HeatmapChartConfig`

- `Title`: The title of the chart.
- `XAxisName`: Optional: Name for the X-axis.
- `YAxisName`: Optional: Name for the Y-axis.
- `Data`: The data for the heatmap. Supported types:
  - `[][]float64`
  - `*insyra.DataTable`
  - `insyra.IDataTable`
- `XAxis`: Optional: X-axis coordinates. If not provided, will use indices.
- `YAxis`: Optional: Y-axis coordinates. If not provided, will use indices.
- `Colors`: Optional: Number of colors in the palette. Default is 20.
- `Alpha`: Optional: Alpha (transparency) for colors. Default is 1.0.

#### `CreateHeatmapChart(config HeatmapChartConfig) *plot.Plot`

Creates a heatmap based on the provided configuration.

#### Heatmap Example

![heatmap_example](./img/gplot_heatmap_example.png)

### Saving Charts

`func SaveChart(plt *plot.Plot, filename string)`

Saves the plot to a file. Supported file formats: `.jpg`, `.jpeg`, `.pdf`, `.png`, `.svg`, `.tex`, `.tif`, `.tiff`
Automatically determine the file extension based on the filename.
