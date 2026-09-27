# [ gplot ] Package

The `gplot` package creates static charts using the Gonum plotting library. It's ideal for generating publication-quality charts that can be saved as PNG, PDF, SVG, or other image formats.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/gplot
```

## Quick Start

```go
package main

import "github.com/HazelnutParadise/insyra/gplot"

func main() {
    // Create a simple bar chart
    config := gplot.BarChartConfig{
        Title: "Monthly Sales",
        XAxis: []string{"Jan", "Feb", "Mar", "Apr"},
    }
    data := []float64{100, 150, 120, 180}
    plt := gplot.CreateBarChart(config, data)
    if err := gplot.SaveChart(plt, "sales.png"); err != nil {
        log.Fatal(err)
    }
}
```

## Supported Chart Types

| Chart Type | Function | Use Case | Accepts |
| ---------- | -------- | -------- | ------- |
| Bar Chart | `CreateBarChart` | Comparing categories | `[]float64`, `*insyra.DataList`, `insyra.IDataList` |
| Histogram | `CreateHistogram` | Distribution analysis | `[]float64`, `*insyra.DataList`, `insyra.IDataList` |
| Line Chart | `CreateLineChart` | Trends over time | `map[string][]float64`, `[]*insyra.DataList`, `[]insyra.IDataList` |
| Scatter Plot | `CreateScatterPlot` | Correlation analysis | `map[string][][]float64`, `[]*insyra.DataList`, `[]insyra.IDataList` |
| Step Chart | `CreateStepChart` | Discrete changes | `map[string][]float64`, `[]*insyra.DataList`, `[]insyra.IDataList` |
| Function Plot | `CreateFunctionPlot` | Mathematical functions | its second argument, `func(float64) float64` |
| Heatmap | `CreateHeatmapChart` | Matrix visualization | `[][]float64`, `*insyra.DataTable`, `insyra.IDataTable` |

Every constructor except `CreateFunctionPlot` takes `data` as `any` and checks its type at run time, so a type the chart does not accept compiles, logs a warning, and returns `nil`. `CreateScatterPlot` reads each list as alternating x and y values and drops an odd last value with a warning.

## Saving Charts

```go
func SaveChart(plt *plot.Plot, filename string) error
```

**Description:** Saves the chart to a file. The format is determined by the file extension. A write failure (missing directory, no permission, disk full) is returned; it does not end the program. A `nil` chart, which is what a `Create...` function returns when it cannot build one, is refused with an error rather than a panic, and so is a file extension outside the list below.

**Parameters:**

- `plt`: Input value for `plt`. Type: `*plot.Plot`.
- `filename`: File path to use. Type: `string`.

**Returns:**

- `error`: non-nil when the chart could not be written.

**Supported formats:** `.png`, `.jpg`, `.jpeg`, `.pdf`, `.svg`, `.tex`, `.tif`, `.tiff`

```go
if err := gplot.SaveChart(plt, "chart.png"); err != nil {
    log.Fatal(err)
}
```

## Chart Types

### Bar Chart

Creates a bar chart for comparing values across categories.

```go
type BarChartConfig struct {
    Title     string    // Chart title
    XAxis     []string  // Optional: category labels; omitted, the bars are numbered 1, 2, 3, ...
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
    BarWidth  float64   // Optional: Bar width (default: 20)
    ErrorBars []float64 // Optional: Error bar values (if provided, must match data length)
}
```

**Example:**

```go
config := gplot.BarChartConfig{
    Title:     "Quarterly Revenue",
    XAxis:     []string{"Q1", "Q2", "Q3", "Q4"},
    XAxisName: "Quarter",
    YAxisName: "Revenue ($K)",
    BarWidth:  25,
}
data := []float64{250, 300, 280, 350}
plt := gplot.CreateBarChart(config, data)
_ = gplot.SaveChart(plt, "revenue.png")
```

![bar_example](./img/gplot_bar_example.png)

**With Error Bars:**

```go
config := gplot.BarChartConfig{
    Title:     "Experimental Results",
    XAxis:     []string{"A", "B", "C", "D"},
    ErrorBars: []float64{0.5, 0.8, 0.6, 0.9},
}
data := []float64{5.2, 7.8, 6.4, 9.1}
plt := gplot.CreateBarChart(config, data)
_ = gplot.SaveChart(plt, "experiment.png")
```

![bar_errorbars_example](./img/gplot_bar_errorbars_example.png)

### Histogram

Creates a histogram to visualize data distribution.

```go
type HistogramConfig struct {
    Title     string // Chart title
    XAxisName string // Optional: X-axis label
    YAxisName string // Optional: Y-axis label
    Bins      int    // Number of bins; zero or less uses 10
}
```

**Example:**

```go
import "math/rand"

// Generate sample data
data := make([]float64, 1000)
for i := range data {
    data[i] = rand.NormFloat64()*15 + 100 // Normal distribution
}

config := gplot.HistogramConfig{
    Title:     "Score Distribution",
    XAxisName: "Score",
    YAxisName: "Frequency",
    Bins:      20,
}
plt := gplot.CreateHistogram(config, data)
_ = gplot.SaveChart(plt, "distribution.png")
```

![histogram_example](./img/gplot_histogram_example.png)

### Line Chart

Creates a line chart for visualizing trends.

```go
type LineChartConfig struct {
    Title     string    // Chart title
    XAxis     []float64 // X-axis data
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
}
```

**Example:**

```go
config := gplot.LineChartConfig{
    Title:     "Temperature Trends",
    XAxisName: "Day",
    YAxisName: "Temperature (C)",
}
data := map[string][]float64{
    "City A": {22, 24, 23, 25, 26},
    "City B": {18, 19, 20, 21, 22},
}
plt := gplot.CreateLineChart(config, data)
_ = gplot.SaveChart(plt, "temperature.png")
```

![line_example](./img/gplot_line_example.png)

### Scatter Plot

Creates a scatter plot for correlation analysis.

```go
type ScatterPlotConfig struct {
    Title     string // Chart title
    XAxisName string // Optional: X-axis label
    YAxisName string // Optional: Y-axis label
}
```

**Data format:** For `map[string][][]float64`, each series is a slice of `[x, y]` coordinate pairs.

**Example:**

```go
config := gplot.ScatterPlotConfig{
    Title: "Height vs Weight",
    XAxisName: "Height (cm)",
    YAxisName: "Weight (kg)",
}
data := map[string][][]float64{
    "Male": {
        {170, 70}, {175, 75}, {180, 80}, {168, 68}, {185, 85},
    },
    "Female": {
        {160, 55}, {165, 60}, {158, 52}, {170, 65}, {163, 58},
    },
}
plt := gplot.CreateScatterPlot(config, data)
_ = gplot.SaveChart(plt, "height_weight.png")
```

### Step Chart

Creates a step chart for data that changes at discrete intervals.

```go
type StepChartConfig struct {
    Title     string    // Chart title
    XAxis     []float64 // X-axis data
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
    StepStyle string    // Optional: "pre", "mid", or "post" (default: "post")
}
```

**Step Styles:**

- `"pre"`: Step before the point (vertical then horizontal)
- `"mid"`: Step at midpoint
- `"post"`: Step after the point (horizontal then vertical)

**Example:**

```go
config := gplot.StepChartConfig{
    Title: "Stock Price Changes",
    XAxis: []float64{9, 10, 11, 12, 13}, // numeric X values
    XAxisName: "Time",
    YAxisName: "Price ($)",
    StepStyle: "post",
}
data := map[string][]float64{
    "Stock A": {100, 102, 101, 105, 103},
}
plt := gplot.CreateStepChart(config, data)
_ = gplot.SaveChart(plt, "stock.png")
```

![step_example](./img/gplot_step_example.png)

### Function Plot

Plots mathematical functions.

```go
type FunctionPlotConfig struct {
    Title     string  // Chart title
    XAxisName string  // X-axis label
    YAxisName string  // Y-axis label
    XMin      float64 // Optional: Minimum X value
    XMax      float64 // Optional: Maximum X value
    YMin      float64 // Optional: Minimum Y value
    YMax      float64 // Optional: Maximum Y value
}
```

When `XMin` and `XMax` are both zero the range is `[-10, 10]`. When `YMin` and `YMax` are both zero, the Y range is taken from the sampled values. The function is sampled at 100 points per unit of X range, and at least 2, so the default range draws 2,000 points. A wide range is not refused. Its cost appears when the chart is saved, because every sample is drawn. Measured on an arm64 Mac on 2026-09-27, `SaveChart` took 0.5 s for a ±1,000 range (200,000 points), 9 s for ±10,000 and 92 s for ±100,000, while `CreateFunctionPlot` stayed under 0.1 s. A ±1e6 range is 200,000,000 points. Keep the X range to what the curve needs.

**Example:**

```go
import "math"

config := gplot.FunctionPlotConfig{
    Title: "Sine Wave",
    XAxisName: "x",
    YAxisName: "sin(x)",
    XMin:  -2 * math.Pi,
    XMax:  2 * math.Pi,
}
plt := gplot.CreateFunctionPlot(config, math.Sin)
_ = gplot.SaveChart(plt, "sine.png")

// Custom function
config2 := gplot.FunctionPlotConfig{
    Title:     "Quadratic Function",
    XAxisName: "x",
    YAxisName: "y",
    XMin:      -2,
    XMax:      6,
}
plt2 := gplot.CreateFunctionPlot(config2, func(x float64) float64 {
    return x*x - 4*x + 3
})
gplot.SaveChart(plt2, "quadratic.png")
```

![function_example](./img/gplot_function_example.png)

### Heatmap

Creates a heatmap for matrix visualization.

```go
type HeatmapChartConfig struct {
    Title     string    // Chart title
    XAxis     []float64 // Optional: X-axis coordinates
    YAxis     []float64 // Optional: Y-axis coordinates
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
    Colors    int       // Optional: Number of colors (zero or less means the default of 20)
    Alpha     float64   // Optional: Transparency (default: 1.0)
}
```

Every row of the data must hold the same number of values. A ragged grid is
refused: the error names the first row whose length differs, and the function
returns `nil`.

**Example:**

```go
// Create correlation matrix data
data := [][]float64{
    {1.0, 0.8, 0.3},
    {0.8, 1.0, 0.5},
    {0.3, 0.5, 1.0},
}

config := gplot.HeatmapChartConfig{
    Title:  "Correlation Matrix",
    XAxis:  []float64{0, 1, 2},
    YAxis:  []float64{0, 1, 2},
    Colors: 20,
}
plt := gplot.CreateHeatmapChart(config, data)
_ = gplot.SaveChart(plt, "correlation.png")
```

![heatmap_example](./img/gplot_heatmap_example.png)

## Using with DataList and DataTable

All chart types support Insyra data structures:

```go
import (
    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/gplot"
)

// Using DataList for bar chart
dl := insyra.NewDataList(100, 150, 120, 180)
config := gplot.BarChartConfig{
    Title: "Sales Data",
    XAxis: []string{"Q1", "Q2", "Q3", "Q4"},
}
plt := gplot.CreateBarChart(config, dl)

// Using DataTable for heatmap
dt := insyra.NewDataTable(
    insyra.NewDataList(1.0, 0.8, 0.3),
    insyra.NewDataList(0.8, 1.0, 0.5),
    insyra.NewDataList(0.3, 0.5, 1.0),
)
heatConfig := gplot.HeatmapChartConfig{
    Title: "Correlation Matrix",
}
plt2 := gplot.CreateHeatmapChart(heatConfig, dt)
```

## A series whose length differs from `XAxis` is dropped

`CreateLineChart` and `CreateStepChart` compare each series with `XAxis`. A series of a different length is skipped with a warning naming it, and the chart is still returned. If every series is dropped, you get a chart with axes and nothing drawn, and it saves without error. When `XAxis` is left out it is generated once: from the longest series for a map, and from the first list for a slice of lists. A shorter map series, or a list whose length differs from the first list, is the one dropped. `CreateBarChart` treats `ErrorBars` of the wrong length the same way: it logs a warning and draws the bars without error bars.

## Tips

- Use meaningful titles and axis labels for better readability
- Choose appropriate bin counts for histograms (typically 10-30)
- For publication, prefer SVG formats for vector graphics
- Error bars should have the same length as the data
