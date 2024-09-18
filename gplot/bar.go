package gplot

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// BarChartConfig defines the configuration for a bar chart.
type BarChartConfig struct {
	Title      string    // Title of the chart.
	Subtitle   string    // Subtitle of the chart (gonum doesn't directly support subtitles, so this can be added as a separate text element).
	XAxis      []string  // X-axis data (categories).
	SeriesData []float64 // Accepts a slice of float64 for bar heights.
	XAxisName  string    // Optional: X-axis name.
	YAxisName  string    // Optional: Y-axis name.
}

// CreateBarChart generates and returns a plot.Plot object based on BarChartConfig.
func CreateBarChart(config BarChartConfig) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Create a Bar plot with the provided data.
	barData := make(plotter.Values, len(config.SeriesData))
	for i, v := range config.SeriesData {
		barData[i] = v
	}

	bars, err := plotter.NewBarChart(barData, vg.Points(20))
	if err != nil {
		panic(err)
	}

	// Set axis labels (categories).
	plt.NominalX(config.XAxis...)

	plt.Add(bars)

	return plt
}
