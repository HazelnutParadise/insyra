# [ gplot ] Package

The `gplot` package creates static charts using the Gonum plotting library. It's ideal for generating publication-quality charts that can be saved as PNG, PDF, SVG, or other image formats.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/gplot
```

## Quick Start

```go
package main

import (
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/gplot"
)

func main() {
    // Create a simple bar chart
    config := gplot.BarChartConfig{
        Title: "Monthly Sales",
        XAxis: []string{"Jan", "Feb", "Mar", "Apr"},
    }
    data := insyra.NewDataList(100, 150, 120, 180)
    plt, err := gplot.CreateBarChart(config, data)
    if err != nil {
        log.Fatal(err)
    }
    if err := gplot.SaveChart(plt, "sales.png"); err != nil {
        log.Fatal(err)
    }
}
```

## Supported Chart Types

| Chart Type | Function | Use Case | Data |
| ---------- | -------- | -------- | ---- |
| Bar Chart | `CreateBarChart` | Comparing categories | `insyra.IDataList` |
| Histogram | `CreateHistogram` | Distribution analysis | `insyra.IDataList` |
| Line Chart | `CreateLineChart` | Trends over time | `...insyra.IDataList` |
| Scatter Plot | `CreateScatterPlot` | Correlation analysis | `...gplot.ScatterSeries` |
| Step Chart | `CreateStepChart` | Discrete changes | `...insyra.IDataList` |
| Function Plot | `CreateFunctionPlot` | Mathematical functions | `func(float64) float64` |
| Heatmap | `CreateHeatmapChart` | Matrix visualization | `insyra.IDataTable` |

Every constructor returns the chart and an `error`. When it cannot build a chart, it returns a `nil` chart and an error that starts with the function's name, such as `gplot: CreateBarChart: the data list is empty`. A failing call logs nothing, so reporting the error is up to you. Each chart's section lists what it refuses.

### From plain slices

The data types are insyra's own, checked by the compiler. Values you hold in plain Go slices are passed like this:

| You have | Pass |
| -------- | ---- |
| a `[]float64` | `insyra.NewDataList(values)` |
| named series for a line or step chart | one `insyra.NewDataList(values).SetName("City A")` per series |
| x and y values for a scatter plot | `gplot.ScatterSeries{Name: "s", X: insyra.NewDataList(xs), Y: insyra.NewDataList(ys)}` |
| a `[][]float64` grid | the table from `insyra.ReadSlice2D(grid)`, one inner slice per row. It takes the number of columns from the first row: a shorter row is padded with `nil`, drawn as 0, and a longer row loses its extra values without a warning, so check that the rows agree first. |

### A cell that is not a number is drawn as 0

Every chart reads its values through `DataList.ToF64Slice`, and the heat map reads each column of the table the same way. A number, a fixed-point decimal included, is drawn as its value. Any other cell is drawn as 0: `nil`, text, a numeric string such as `"2"`, and a `bool`. `NewDataList(1, "2", nil, "abc")` is drawn as the bars 1, 0, 0 and 0, with no warning. Clean a column before charting it: `ParseNumbers` turns numeric text into numbers, and `ClearNilsAndNaNs` removes `nil` and `NaN` cells.

A `NaN` or an infinity is not drawn as 0. The bar chart, the histogram, and the line, step and scatter charts refuse it with an error. The heat map refuses an infinity and a table of `NaN` alone, and draws a `NaN` among numbers as an empty cell.

## Saving Charts

```go
func SaveChart(plt *plot.Plot, filename string) error
```

**Description:** Saves the chart to a file. The format is determined by the file extension. A write failure (missing directory, no permission, disk full) is returned; it does not end the program. A `nil` chart, which is what a `Create...` function returns alongside its error, is refused with an error rather than a panic, and so is a file extension outside the list below.

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
func CreateBarChart(config BarChartConfig, data insyra.IDataList) (*plot.Plot, error)

type BarChartConfig struct {
    Title     string    // Chart title
    XAxis     []string  // Optional: category labels; omitted, the bars are numbered 1, 2, 3, ...
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
    BarWidth  float64   // Optional: Bar width (default: 20)
    ErrorBars []float64 // Optional: Error bar values (if provided, must match data length)
}
```

Returns an error when `data` is `nil` or empty, or holds a `NaN` or an infinity. With `ErrorBars` set, it also returns an error when their number differs from the number of bars or one of them is a `NaN` or an infinity, instead of drawing the bars without them.

**Example:**

```go
config := gplot.BarChartConfig{
    Title:     "Quarterly Revenue",
    XAxis:     []string{"Q1", "Q2", "Q3", "Q4"},
    XAxisName: "Quarter",
    YAxisName: "Revenue ($K)",
    BarWidth:  25,
}
data := insyra.NewDataList(250, 300, 280, 350)
plt, err := gplot.CreateBarChart(config, data)
if err != nil {
    log.Fatal(err)
}
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
data := insyra.NewDataList(5.2, 7.8, 6.4, 9.1)
plt, err := gplot.CreateBarChart(config, data)
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "experiment.png")
```

![bar_errorbars_example](./img/gplot_bar_errorbars_example.png)

### Histogram

Creates a histogram to visualize data distribution.

```go
func CreateHistogram(config HistogramConfig, data insyra.IDataList) (*plot.Plot, error)

type HistogramConfig struct {
    Title     string // Chart title
    XAxisName string // Optional: X-axis label
    YAxisName string // Optional: Y-axis label
    Bins      int    // Number of bins; zero or less uses 10
}
```

Returns an error when `data` is `nil` or empty, or holds a `NaN` or an infinity, which no bin can hold.

**Example:**

```go
import "math/rand"

// Generate sample data
values := make([]float64, 1000)
for i := range values {
    values[i] = rand.NormFloat64()*15 + 100 // Normal distribution
}

config := gplot.HistogramConfig{
    Title:     "Score Distribution",
    XAxisName: "Score",
    YAxisName: "Frequency",
    Bins:      20,
}
plt, err := gplot.CreateHistogram(config, insyra.NewDataList(values))
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "distribution.png")
```

![histogram_example](./img/gplot_histogram_example.png)

### Line Chart

Creates a line chart for visualizing trends. Each list is one line, named after the list in the legend.

```go
func CreateLineChart(config LineChartConfig, data ...insyra.IDataList) (*plot.Plot, error)

type LineChartConfig struct {
    Title     string    // Chart title
    XAxis     []float64 // X-axis data
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
}
```

Returns an error when no list is given, every list is `nil`, or any series cannot be drawn; that error names each such series and why (see [A series that cannot be drawn fails the chart](#a-series-that-cannot-be-drawn-fails-the-chart)). A `nil` list among real ones is skipped with a warning.

**Example:**

```go
config := gplot.LineChartConfig{
    Title:     "Temperature Trends",
    XAxisName: "Day",
    YAxisName: "Temperature (C)",
}
plt, err := gplot.CreateLineChart(config,
    insyra.NewDataList(22, 24, 23, 25, 26).SetName("City A"),
    insyra.NewDataList(18, 19, 20, 21, 22).SetName("City B"),
)
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "temperature.png")
```

![line_example](./img/gplot_line_example.png)

### Scatter Plot

Creates a scatter plot for correlation analysis.

```go
func CreateScatterPlot(config ScatterPlotConfig, series ...ScatterSeries) (*plot.Plot, error)

type ScatterPlotConfig struct {
    Title     string // Chart title
    XAxisName string // Optional: X-axis label
    YAxisName string // Optional: Y-axis label
}

// ScatterSeries is one set of points: point i is (X[i], Y[i]).
type ScatterSeries struct {
    Name string           // Label in the legend
    X    insyra.IDataList // X coordinates
    Y    insyra.IDataList // Y coordinates
}
```

Two columns of a table make one series. Returns an error when no series is given, when a series has a `nil` `X` or `Y`, when a series' `X` and `Y` differ in length, or when a series has no points or holds a `NaN` or an infinity. The error names the series and why.

**Example:**

```go
config := gplot.ScatterPlotConfig{
    Title:     "Height vs Weight",
    XAxisName: "Height (cm)",
    YAxisName: "Weight (kg)",
}
male := insyra.NewDataTable(
    insyra.NewDataList(170, 175, 180, 168, 185).SetName("height"),
    insyra.NewDataList(70, 75, 80, 68, 85).SetName("weight"),
)
plt, err := gplot.CreateScatterPlot(config,
    gplot.ScatterSeries{Name: "Male", X: male.GetColByName("height"), Y: male.GetColByName("weight")},
    gplot.ScatterSeries{
        Name: "Female",
        X:    insyra.NewDataList(160, 165, 158, 170, 163),
        Y:    insyra.NewDataList(55, 60, 52, 65, 58),
    },
)
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "height_weight.png")
```

### Step Chart

Creates a step chart for data that changes at discrete intervals. Each list is one step line, named after the list in the legend.

```go
func CreateStepChart(config StepChartConfig, data ...insyra.IDataList) (*plot.Plot, error)

type StepChartConfig struct {
    Title     string    // Chart title
    XAxis     []float64 // X-axis data
    XAxisName string    // Optional: X-axis label
    YAxisName string    // Optional: Y-axis label
    StepStyle string    // Optional: "pre", "mid", or "post" (default: "post")
}
```

Returns an error in the same cases as `CreateLineChart`, and when `StepStyle` is anything other than `"pre"`, `"mid"`, `"post"` or empty, so a misspelled style is reported instead of being drawn as `"post"`.

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
plt, err := gplot.CreateStepChart(config, insyra.NewDataList(100, 102, 101, 105, 103).SetName("Stock A"))
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "stock.png")
```

![step_example](./img/gplot_step_example.png)

### Function Plot

Plots mathematical functions.

```go
func CreateFunctionPlot(config FunctionPlotConfig, function func(x float64) float64) (*plot.Plot, error)

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

Returns an error when `function` is `nil`, or when `XMin`, `XMax`, `YMin` or `YMax` is a `NaN` or an infinity: an infinite range would ask for an unbounded number of samples. When `XMin` and `XMax` are both zero the range is `[-10, 10]`. When `YMin` and `YMax` are both zero, the Y range is taken from the sampled values. The function is sampled at 100 points per unit of X range, and at least 2, so the default range draws 2,000 points. A wide range is not refused. Its cost appears when the chart is saved, because every sample is drawn. Measured on an arm64 Mac on 2026-09-27, `SaveChart` took 0.5 s for a ±1,000 range (200,000 points), 9 s for ±10,000 and 92 s for ±100,000, while `CreateFunctionPlot` stayed under 0.1 s. A ±1e6 range is 200,000,000 points. Keep the X range to what the curve needs.

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
plt, err := gplot.CreateFunctionPlot(config, math.Sin)
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "sine.png")

// Custom function
config2 := gplot.FunctionPlotConfig{
    Title:     "Quadratic Function",
    XAxisName: "x",
    YAxisName: "y",
    XMin:      -2,
    XMax:      6,
}
plt2, err := gplot.CreateFunctionPlot(config2, func(x float64) float64 {
    return x*x - 4*x + 3
})
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt2, "quadratic.png")
```

![function_example](./img/gplot_function_example.png)

### Heatmap

Creates a heatmap for matrix visualization. Row *i* of the table is row *i* of the grid, and column *j* is column *j*.

```go
func CreateHeatmapChart(config HeatmapChartConfig, data insyra.IDataTable) (*plot.Plot, error)

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

Returns an error when `data` is `nil`, has no rows or no columns, holds an infinity, or holds nothing but `NaN`; the error names the first infinite cell by its row and column, counted from 0. A `NaN` among numbers is drawn as an empty cell. A table keeps its columns the same length by padding a shorter one with `nil`, and that padding is drawn as 0.

**Example:**

```go
// A correlation matrix as a grid, one inner slice per row
grid, err := insyra.ReadSlice2D([][]float64{
    {1.0, 0.8, 0.3},
    {0.8, 1.0, 0.5},
    {0.3, 0.5, 1.0},
})
if err != nil {
    log.Fatal(err)
}

config := gplot.HeatmapChartConfig{
    Title:  "Correlation Matrix",
    XAxis:  []float64{0, 1, 2},
    YAxis:  []float64{0, 1, 2},
    Colors: 20,
}
plt, err := gplot.CreateHeatmapChart(config, grid)
if err != nil {
    log.Fatal(err)
}
_ = gplot.SaveChart(plt, "correlation.png")
```

The table returned by `stats.CorrelationMatrix` can be passed as it is.

![heatmap_example](./img/gplot_heatmap_example.png)

## A series that cannot be drawn fails the chart

`CreateLineChart`, `CreateStepChart` and `CreateScatterPlot` draw every series they are given, or none. A line or step series whose length differs from `XAxis`, an empty series, and a series holding a `NaN` or an infinity each make the call return a `nil` chart and an error, so a chart never comes back missing a series you asked for. When `XAxis` is left out, it is generated from the first list's length, so every other list has to match the first. The error names every series that failed and why, for example `gplot: CreateLineChart: cannot draw every series: series "two" has 2 values but XAxis has 3`, and nothing is logged.

A `nil` list among real ones is different: it is missing input rather than a series, so the line and step charts drop it with a warning and draw the rest. It is the only case in `gplot` that is reported as a warning rather than an error.

## Tips

- Use meaningful titles and axis labels for better readability
- Choose appropriate bin counts for histograms (typically 10-30)
- For publication, prefer SVG formats for vector graphics
