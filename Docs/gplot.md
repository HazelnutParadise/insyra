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
    gplot.SaveChart(plt, "sales.png")
}
```

## Supported Chart Types

Every `Create...` function here takes `data` as an `any` and type-switches on
it, so the accepted types differ per chart and an unsupported one is a runtime
`nil` rather than a compile error.

| Chart Type | Function | Use Case | Accepts |
| ---------- | -------- | -------- | ------- |
| Bar Chart | `CreateBarChart` | Comparing categories | `[]float64`, `*insyra.DataList`, `insyra.IDataList` |
| Histogram | `CreateHistogram` | Distribution analysis | `[]float64`, `*insyra.DataList`, `insyra.IDataList` |
| Line Chart | `CreateLineChart` | Trends over time | `map[string][]float64`, `[]*insyra.DataList`, `[]insyra.IDataList` |
| Scatter Plot | `CreateScatterPlot` | Correlation analysis | `map[string][][]float64`, `[]*insyra.DataList`, `[]insyra.IDataList` |
| Step Chart | `CreateStepChart` | Discrete changes | `map[string][]float64`, `[]*insyra.DataList`, `[]insyra.IDataList` |
| Function Plot | `CreateFunctionPlot` | Mathematical functions | `func(float64) float64` (its second argument, not a data structure) |
| Heatmap | `CreateHeatmapChart` | Matrix visualization | `[][]float64`, `*insyra.DataTable`, `insyra.IDataTable` |

The single-series charts take one list; the multi-series ones take a map or a
slice of lists and read the series name from the map key or the list's name.
Anything else — including a `[]insyra.IDataList` where the chart wants a single
one, or a `[]*insyra.DataTable` where the heatmap wants a single one — is
refused with a warning and a `nil` return.

## Saving Charts

```go
func SaveChart(plt *plot.Plot, filename string)
```

**Description:** Saves the chart to a file. The format is determined by the file extension. A chart that a `Create...` function refused to build is a `nil` `*plot.Plot`, which `SaveChart` cannot render — check for `nil` before saving.

**Parameters:**

- `plt`: Input value for `plt`. Type: `*plot.Plot`.
- `filename`: File path to use. Type: `string`.

**Returns:**

- None. A path that cannot be written, or an extension that is not supported, goes to `insyra.LogFatal`, which **ends the program** with status 1 unless `insyra.Config.SetDontPanic(true)` is set, in which case it is only logged. Create the output directory before calling this.

**Supported formats:** `.png`, `.jpg`, `.jpeg`, `.pdf`, `.svg`, `.tex`, `.tif`, `.tiff`

```go
gplot.SaveChart(plt, "chart.png")  // PNG format
gplot.SaveChart(plt, "chart.pdf")  // PDF format
gplot.SaveChart(plt, "chart.svg")  // SVG format
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
gplot.SaveChart(plt, "revenue.png")
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
gplot.SaveChart(plt, "experiment.png")
```

![bar_errorbars_example](./img/gplot_bar_errorbars_example.png)

### Histogram

Creates a histogram to visualize data distribution.

```go
type HistogramConfig struct {
    Title     string // Chart title
    XAxisName string // Optional: X-axis label
    YAxisName string // Optional: Y-axis label
    Bins      int    // Number of bins; zero or negative means 10
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
gplot.SaveChart(plt, "distribution.png")
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
gplot.SaveChart(plt, "temperature.png")
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
gplot.SaveChart(plt, "height_weight.png")
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
gplot.SaveChart(plt, "stock.png")
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

`CreateFunctionPlot` samples the function to draw it, and the sample count is
fixed at **100 points per unit of X range** — `int((XMax - XMin) * 100)`, with
a floor of 2. Leaving both `XMin` and `XMax` at zero means the range `[-10, 10]`
and 2,000 samples. Every other combination is your responsibility: `XMin: -1e6,
XMax: 1e6` asks for 200,000,000 points. Nothing rejects that and nothing errors;
the function is just evaluated once per point to work out the Y range and the
plot then carries that many points, so a wide window turns into a very slow
call rather than an error. Measured on an M3, a 200,000-point window took 2 ms
and a 2,000,000-point window 14 ms. Keep the window to what the shape actually
needs, and tighten it with `YMin`/`YMax` rather than widening `XMin`/`XMax`.

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
gplot.SaveChart(plt, "sine.png")

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

No row of the data may hold fewer values than the first row. A grid with a
shorter row is refused: a warning names that row and the function returns
`nil`. Values in a row past the first row's length are ignored.

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
gplot.SaveChart(plt, "correlation.png")
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

## Tips

- Use meaningful titles and axis labels for better readability
- Choose appropriate bin counts for histograms (typically 10-30)
- For publication, prefer SVG formats for vector graphics
- Error bars should have the same length as the data

## Things to be careful about

### A series that does not match `XAxis` is dropped, and you still get a chart

`CreateLineChart` and `CreateStepChart` check each series against the shared
`XAxis` one at a time. A series whose length differs is skipped with a warning
naming it, and the rest are drawn — the constructor still returns a chart. If
every series is dropped, the result is not `nil`: it is a real `*plot.Plot`
with axes and an empty drawing area, which saves to a perfectly valid file that
shows nothing. When `XAxis` is left nil it is generated once for the whole
chart, from the longest series for a map and from the first list otherwise, so
a shorter series in the same map is what usually trips this.

`CreateBarChart` treats a mismatched `ErrorBars` the same way: the lengths are
compared, a warning names both, the error bars are left off, and the bars are
drawn as usual.
