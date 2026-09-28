// gplot/scatter.go

package gplot

import (
	"fmt"
	"image/color"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ScatterPlotConfig defines the configuration for a multi-series scatter plot.
type ScatterPlotConfig struct {
	Title     string // Title of the chart.
	XAxisName string // Optional: X-axis name.
	YAxisName string // Optional: Y-axis name.
}

// ScatterSeries is one set of points for CreateScatterPlot: point i is
// (X[i], Y[i]), and Name labels the series in the legend. Two columns of a
// table are one series: ScatterSeries{Name: "s", X: dt.GetColByName("x"),
// Y: dt.GetColByName("y")}.
type ScatterSeries struct {
	Name string
	X    insyra.IDataList
	Y    insyra.IDataList
}

// CreateScatterPlot draws one set of points per series.
//
// It returns a nil chart and an error when no series is given, when a series
// has a nil X or Y, when a series' X and Y differ in length, or when a series
// cannot be drawn because it has no points or holds a NaN or an infinity. The
// error names the series and why.
//
// The values are read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is drawn as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is drawn as 0.
func CreateScatterPlot(config ScatterPlotConfig, series ...ScatterSeries) (*plot.Plot, error) {
	if len(series) == 0 {
		return nil, chartError("CreateScatterPlot", "no data to draw")
	}
	// Each list is read once, so the lengths checked are the lengths drawn.
	xs := make([][]float64, len(series))
	ys := make([][]float64, len(series))
	for i, s := range series {
		if isNilList(s.X) || isNilList(s.Y) {
			return nil, chartError("CreateScatterPlot", "series %d (%q) needs both X and Y", i, s.Name)
		}
		xs[i], ys[i] = readValues(s.X), readValues(s.Y)
		if len(xs[i]) != len(ys[i]) {
			return nil, chartError("CreateScatterPlot", "series %d (%q) has %d X values and %d Y values", i, s.Name, len(xs[i]), len(ys[i]))
		}
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	var failed []string
	for i, s := range series {
		if err := addScatterSeries(plt, s.Name, xs[i], ys[i], i); err != nil {
			failed = append(failed, err.Error())
		}
	}
	if err := seriesError("CreateScatterPlot", failed); err != nil {
		return nil, err
	}
	return plt, nil
}

// addScatterSeries adds one series to the plot, or says why it cannot.
func addScatterSeries(plt *plot.Plot, seriesName string, xs, ys []float64, index int) error {
	if len(xs) == 0 {
		return fmt.Errorf("series %q has no points", seriesName)
	}
	scatterData := make(plotter.XYs, len(xs))
	for i := range xs {
		scatterData[i].X = xs[i]
		scatterData[i].Y = ys[i]
	}

	scatter, err := plotter.NewScatter(scatterData)
	if err != nil {
		return fmt.Errorf("series %q cannot be drawn: %w", seriesName, err)
	}

	// Set different colors and shapes for each series
	colors := []color.Color{
		color.RGBA{R: 255, G: 0, B: 0, A: 255},     // Red
		color.RGBA{R: 0, G: 0, B: 255, A: 255},     // Blue
		color.RGBA{R: 0, G: 128, B: 0, A: 255},     // Green
		color.RGBA{R: 255, G: 165, B: 0, A: 255},   // Orange
		color.RGBA{R: 128, G: 0, B: 128, A: 255},   // Purple
		color.RGBA{R: 0, G: 128, B: 128, A: 255},   // Teal
		color.RGBA{R: 255, G: 192, B: 203, A: 255}, // Pink
	}

	scatter.Color = colors[index%len(colors)]

	// Set different shapes for each series
	shapes := []draw.GlyphDrawer{
		draw.CircleGlyph{},
		draw.SquareGlyph{},
		draw.TriangleGlyph{},
		draw.PlusGlyph{},
		draw.CrossGlyph{},
	}

	scatter.Shape = shapes[index%len(shapes)]
	scatter.Radius = vg.Points(4)

	// Add the scatter plot to the chart
	plt.Add(scatter)
	plt.Legend.Add(seriesName, scatter)
	return nil
}
