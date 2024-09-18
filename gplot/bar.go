// gplot/bar.go

package gplot

import (
	"github.com/HazelnutParadise/insyra" // 確保這是正確的導入路徑
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// BarChartConfig defines the configuration for a single series bar chart.
type BarChartConfig struct {
	Title      string   // Title of the chart.
	XAxis      []string // X-axis data (categories).
	SeriesData any      // Accepts []float64 or *insyra.DataList
	XAxisName  string   // Optional: X-axis name.
	YAxisName  string   // Optional: Y-axis name.
	BarWidth   float64  // Optional: Bar width for each bar in the chart. Default is 20.
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

	// Determine the type of SeriesData and handle it accordingly
	switch data := config.SeriesData.(type) {
	case []float64:
		values = data
	case *insyra.DataList:
		values = data.ToF64Slice()
	case insyra.IDataList:
		values = data.ToF64Slice()
	default:
		insyra.LogWarning("Unsupported SeriesData type: %T\n", config.SeriesData)
		return nil
	}

	// Create a Bar plot with the processed values.
	barData := make(plotter.Values, len(values))
	for i, v := range values {
		barData[i] = v
	}

	barWidth := config.BarWidth
	if barWidth == 0 {
		barWidth = 20 // Default bar width
	}

	bars, err := plotter.NewBarChart(barData, vg.Points(barWidth))
	if err != nil {
		panic(err)
	}

	// Set axis labels (categories).
	plt.NominalX(config.XAxis...)

	plt.Add(bars)

	return plt
}
