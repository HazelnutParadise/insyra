// gplot/bar.go

package gplot

import (
	"strconv"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// BarChartConfig defines the configuration for a single series bar chart.
type BarChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []string  // X-axis data (categories).
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
	BarWidth  float64   // Optional: Bar width for each bar in the chart. Default is 20.
	ErrorBars []float64 // Optional: Error bar values for each bar. If provided, must match the length of Data.
}

// CreateBarChart draws one bar per value in data. A plain slice is passed as
// insyra.NewDataList(values).
//
// The values are read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is drawn as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is drawn as 0.
//
// It returns a nil chart and an error when data is nil or empty, or holds a
// NaN or an infinity, and when config.ErrorBars is given but its length
// differs from the data's or it holds a NaN or an infinity.
func CreateBarChart(config BarChartConfig, data insyra.IDataList) (*plot.Plot, error) {
	if isNilList(data) {
		return nil, chartError("CreateBarChart", "no data to draw")
	}
	values := readValues(data)
	if len(values) == 0 {
		return nil, chartError("CreateBarChart", "the data list is empty")
	}

	barWidth := config.BarWidth
	if barWidth == 0 {
		barWidth = 20 // Default bar width
	}

	bars, err := plotter.NewBarChart(plotter.Values(values), vg.Points(barWidth))
	if err != nil {
		return nil, chartError("CreateBarChart", "cannot draw the bars: %w", err)
	}

	// Error bars that were asked for are drawn, or the call fails: a chart
	// without them would look complete while missing what was requested.
	var errBars *plotter.YErrorBars
	if len(config.ErrorBars) > 0 {
		if len(config.ErrorBars) != len(values) {
			return nil, chartError("CreateBarChart", "ErrorBars has %d values but the data has %d", len(config.ErrorBars), len(values))
		}
		errBars, err = plotter.NewYErrorBars(&barErrorData{values: values, errorBars: config.ErrorBars})
		if err != nil {
			return nil, chartError("CreateBarChart", "cannot draw the error bars: %w", err)
		}
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Set axis labels (categories). gonum's NominalX indexes names[0] with no
	// length check, so an empty XAxis — which is what a zero-value config has —
	// used to panic here. Number the bars instead, the way plot.CreateBarChart
	// does, so a config without labels still gives a readable chart.
	labels := config.XAxis
	if len(labels) == 0 {
		labels = make([]string, len(values))
		for i := range labels {
			labels[i] = strconv.Itoa(i + 1)
		}
	}
	plt.NominalX(labels...)

	plt.Add(bars)
	if errBars != nil {
		plt.Add(errBars)
	}

	return plt, nil
}

// barErrorData implements the XYer and YErrorer interfaces for bar chart error bars
type barErrorData struct {
	values    []float64
	errorBars []float64
}

func (d *barErrorData) Len() int {
	return len(d.values)
}

func (d *barErrorData) XY(i int) (float64, float64) {
	return float64(i), d.values[i]
}

func (d *barErrorData) YError(i int) (float64, float64) {
	// Return symmetric error bars (low and high are the same)
	return d.errorBars[i], d.errorBars[i]
}
