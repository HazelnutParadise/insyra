// gplot/scatter.go

package gplot

import (
	"fmt"
	"image/color"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// ScatterChartConfig defines the configuration for a multi-series scatter chart.
type ScatterChartConfig struct {
	Title     string   // Title of the chart.
	XAxisName string   // Optional: X-axis name.
	YAxisName string   // Optional: Y-axis name.
	Colors    []string // Optional: Series colours in place of the shared palette, as #RGB, #RRGGBB, #RRGGBBAA or a CSS name. A short list repeats.
}

// ScatterSeries is one set of points for CreateScatterChart: point i is
// (X[i], Y[i]), and Name labels the series in the legend. Two columns of a
// table are one series: ScatterSeries{Name: "s", X: dt.GetColByName("x"),
// Y: dt.GetColByName("y")}.
type ScatterSeries struct {
	Name string
	X    insyra.IDataList
	Y    insyra.IDataList
}

// CreateScatterChart draws one set of points per series, in argument order.
//
// It returns a nil chart and an error when no series is given, or when a
// series has a nil X or Y, X and Y of different lengths, no points, or a cell
// that is not a Go number (nil, text, a numeric string such as "2", or a
// bool), a NaN or an infinity. The error names every such series by its
// position, counting from one, and its name, and why.
func CreateScatterChart(config ScatterChartConfig, series ...ScatterSeries) (*plot.Plot, error) {
	if len(series) == 0 {
		return nil, chartError("CreateScatterChart", "no data to draw")
	}
	colors, err := readColors("CreateScatterChart", config.Colors)
	if err != nil {
		return nil, err
	}
	xs := make([][]float64, len(series))
	ys := make([][]float64, len(series))
	var failed []string
	for i, s := range series {
		var err error
		if xs[i], ys[i], err = readScatterSeries(i, s); err != nil {
			failed = append(failed, err.Error())
		}
	}
	if err := seriesError("CreateScatterChart", failed); err != nil {
		return nil, err
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	for i, s := range series {
		if err := addScatterSeries(plt, s.Name, xs[i], ys[i], i, colors.at(i)); err != nil {
			failed = append(failed, err.Error())
		}
	}
	if err := seriesError("CreateScatterChart", failed); err != nil {
		return nil, err
	}
	return plt, nil
}

// readScatterSeries reads one series' X and Y, or says why it cannot be drawn.
// i is the series' place among the arguments, counting from zero. The lengths
// are checked on the copies it reads, so a list changed by another goroutine
// in between gives an error instead of a mismatched pair.
func readScatterSeries(i int, s ScatterSeries) ([]float64, []float64, error) {
	if isNilList(s.X) || isNilList(s.Y) {
		return nil, nil, fmt.Errorf("series %d (%q) needs both X and Y", i+1, s.Name)
	}
	xCells, yCells := s.X.Data(), s.Y.Data()
	if len(xCells) != len(yCells) {
		return nil, nil, fmt.Errorf("series %d (%q) has %d X values and %d Y values", i+1, s.Name, len(xCells), len(yCells))
	}
	if len(xCells) == 0 {
		return nil, nil, fmt.Errorf("series %d (%q) has no points", i+1, s.Name)
	}
	x, err := readScatterAxis(i, s.Name, "X", xCells)
	if err != nil {
		return nil, nil, err
	}
	y, err := readScatterAxis(i, s.Name, "Y", yCells)
	if err != nil {
		return nil, nil, err
	}
	return x, y, nil
}

// readScatterAxis reads every cell of dl as a number. A cell that is not a Go
// number would be a point drawn in the wrong place, so it is refused instead
// of read as 0 the way the other charts read it. The cell is quoted in the
// error, so control characters in data from a file reach a terminal escaped.
func readScatterAxis(i int, name, axis string, cells []any) ([]float64, error) {
	out := make([]float64, len(cells))
	for row, cell := range cells {
		f, ok := insyra.ToFloat64Safe(cell)
		if !ok {
			return nil, fmt.Errorf("series %d (%q): %s at row index %d is %q (%T), not a number", i+1, name, axis, row, fmt.Sprint(cell), cell)
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("series %d (%q): %s at row index %d is %v; a chart cannot draw NaN or an infinity", i+1, name, axis, row, f)
		}
		out[row] = f
	}
	return out, nil
}

// addScatterSeries adds one series to the plot, or says why it cannot.
func addScatterSeries(plt *plot.Plot, seriesName string, xs, ys []float64, index int, pointColor color.Color) error {
	scatterData := make(plotter.XYs, len(xs))
	for i := range xs {
		scatterData[i].X = xs[i]
		scatterData[i].Y = ys[i]
	}

	scatter, err := plotter.NewScatter(scatterData)
	if err != nil {
		return fmt.Errorf("series %q cannot be drawn: %w", seriesName, err)
	}

	scatter.Color = pointColor

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
