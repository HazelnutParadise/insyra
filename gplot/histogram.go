package gplot

import (
	"github.com/HazelnutParadise/insyra" // 確保這是正確的導入路徑
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

// HistogramConfig defines the configuration for a single series histogram.
type HistogramConfig struct {
	Title     string // Title of the chart.
	Data      any    // Accepts []float64 or *insyra.DataList or insyra.IDataList.
	XAxisName string // Optional: X-axis name.
	YAxisName string // Optional: Y-axis name.
	Bins      int    // Number of bins for the histogram.
}

// CreateHistogram generates and returns a plot.Plot object based on HistogramConfig.
func CreateHistogram(config HistogramConfig) *plot.Plot {
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
		insyra.LogWarning("Unsupported Data type: %T\n", config.Data)
		return nil
	}

	// Create the histogram.
	hist, err := plotter.NewHist(plotter.Values(values), config.Bins)
	if err != nil {
		panic(err)
	}

	// Add the histogram to the plot.
	plt.Add(hist)

	return plt
}
