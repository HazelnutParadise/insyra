// gplot/line.go

package gplot

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// LineChartConfig defines the configuration for a multi-series line chart.
type LineChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []float64 // X-axis data.
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
}

// CreateLineChart draws one line per list, named after the list. A plain
// slice is passed as insyra.NewDataList(values).SetName("name").
//
// When config.XAxis is nil, it is 0, 1, 2, ... up to the first list's length.
// It returns a nil chart and an error when no list is given, every one is
// nil, or any list cannot be drawn: its length differs from XAxis, it is
// empty, or it holds a NaN or an infinity. That error names each such list and
// why. A nil list among real ones is skipped with a warning.
//
// The values are read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is drawn as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is drawn as 0.
func CreateLineChart(config LineChartConfig, data ...insyra.IDataList) (*plot.Plot, error) {
	return drawLines("CreateLineChart", config.Title, config.XAxis, config.XAxisName, config.YAxisName, plotter.NoStep, data)
}

// drawLines builds a line or step chart; stepKind is plotter.NoStep for a line.
func drawLines(funcName, title string, xAxis []float64, xAxisName, yAxisName string, stepKind plotter.StepKind, data []insyra.IDataList) (*plot.Plot, error) {
	data = nonNilLists(funcName, data)
	if len(data) == 0 {
		return nil, chartError(funcName, "no data to draw")
	}

	plt := plot.New()
	plt.Title.Text = title
	plt.X.Label.Text = xAxisName
	plt.Y.Label.Text = yAxisName

	if xAxis == nil {
		xAxis = make([]float64, data[0].Len())
		for i := range xAxis {
			xAxis[i] = float64(i)
		}
	}

	var failed []string
	for i, dl := range data {
		if err := addLineSeries(plt, dl.GetName(), readValues(dl), xAxis, i, stepKind); err != nil {
			failed = append(failed, err.Error())
		}
	}
	if err := seriesError(funcName, failed); err != nil {
		return nil, err
	}
	return plt, nil
}

// seriesError is the error for a chart that could not draw every series it
// was given, naming each and why, or nil when every series was drawn. A chart
// missing a series is not returned: the caller asked for all of them.
func seriesError(funcName string, failed []string) error {
	if len(failed) == 0 {
		return nil
	}
	return chartError(funcName, "cannot draw every series: %s", strings.Join(failed, "; "))
}

// addLineSeries adds one series to the plot, or says why it cannot.
func addLineSeries(plt *plot.Plot, seriesName string, values []float64, xAxis []float64, index int, stepKind plotter.StepKind) error {
	if len(xAxis) != len(values) {
		return fmt.Errorf("series %q has %d values but XAxis has %d", seriesName, len(values), len(xAxis))
	}
	if len(values) == 0 {
		return fmt.Errorf("series %q has no values", seriesName)
	}

	lineData := make(plotter.XYs, len(xAxis))
	for j := range xAxis {
		lineData[j].X = xAxis[j]
		lineData[j].Y = values[j]
	}

	line, err := plotter.NewLine(lineData)
	if err != nil {
		return fmt.Errorf("series %q cannot be drawn: %w", seriesName, err)
	}
	line.StepStyle = stepKind

	// Set different line styles for each series
	switch index % 5 {
	case 0:
		// 實線
		line.LineStyle = plotter.DefaultLineStyle
	case 1:
		// 長虛線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(8), vg.Points(4)}
	case 2:
		// 點線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(2), vg.Points(2)}
	case 3:
		// 短虛線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(4), vg.Points(2)}
	case 4:
		// 交替虛線和實線
		line.LineStyle = plotter.DefaultLineStyle
		line.Dashes = []vg.Length{vg.Points(6), vg.Points(2), vg.Points(1), vg.Points(2)}
	}

	plt.Add(line)
	plt.Legend.Add(seriesName, line)
	return nil
}
