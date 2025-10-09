// gplot/bar.go

package gplot

import (
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// BarChartConfig defines the configuration for a single series bar chart.
type BarChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []string  // X-axis data (categories).
	Data      any       // Accepts []float64 or *insyra.DataList
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
	BarWidth  float64   // Optional: Bar width for each bar in the chart. Default is 20.
	ErrorBars []float64 // Optional: Error bar values for each bar. If provided, must match the length of Data.
}

// CreateBarChart generates and returns a plot.Plot object based on BarChartConfig.
func CreateBarChart(config BarChartConfig) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	var values []float64

	// Determine the type of Data and handle it accordingly
	switch data := config.Data.(type) {
	case []float64:
		values = data
	case *insyra.DataList:
		values = data.ToF64Slice()
	case insyra.IDataList:
		values = data.ToF64Slice()
	default:
		insyra.LogWarning("gplot", "CreateBarChart", "Unsupported Data type: %T\n", config.Data)
		return nil
	}

	// Create a Bar plot with the processed values.
	barData := make(plotter.Values, len(values))
	copy(barData, values)

	barWidth := config.BarWidth
	if barWidth == 0 {
		barWidth = 20 // Default bar width
	}

	bars, err := plotter.NewBarChart(barData, vg.Points(barWidth))
	if err != nil {
		insyra.LogWarning("gplot", "CreateBarChart", "failed to create bar chart: %v", err)
		return nil
	}

	// Set axis labels (categories).
	plt.NominalX(config.XAxis...)

	plt.Add(bars)

	// Add error bars if provided
	if len(config.ErrorBars) > 0 {
		if len(config.ErrorBars) != len(values) {
			insyra.LogWarning("gplot", "CreateBarChart", "ErrorBars length (%d) does not match Data length (%d)", len(config.ErrorBars), len(values))
		} else {
			// Create a custom XYError data structure for error bars
			errData := &barErrorData{
				values:    values,
				errorBars: config.ErrorBars,
			}

			errBars, err := plotter.NewYErrorBars(errData)
			if err != nil {
				insyra.LogWarning("gplot", "CreateBarChart", "failed to create error bars: %v", err)
			} else {
				plt.Add(errBars)
			}
		}
	}

	return plt
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
