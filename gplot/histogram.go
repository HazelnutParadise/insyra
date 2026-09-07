package gplot

import (
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

// HistogramConfig defines the configuration for a single series histogram.
type HistogramConfig struct {
	Title     string // Title of the chart.
	XAxisName string // Optional: X-axis name.
	YAxisName string // Optional: Y-axis name.
	Bins      int    // Number of bins for the histogram. Zero or negative means the default (10).
}

// CreateHistogram generates and returns a plot.Plot object based on HistogramConfig.
// The Data field can be of type []float64, *insyra.DataList, or insyra.IDataList.
// defaultHistogramBins is used when HistogramConfig.Bins is left at its zero
// value, so a zero-value config still produces a chart.
const defaultHistogramBins = 10

func CreateHistogram(config HistogramConfig, data any) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	var values []float64

	// Determine the type of Data and handle it accordingly
	switch data := data.(type) {
	case []float64:
		values = data
	case *insyra.DataList:
		values = data.ToF64Slice()
	case insyra.IDataList:
		values = data.ToF64Slice()
	default:
		insyra.LogWarning("gplot", "CreateHistogram", "Unsupported Data type: %T\n", data)
		return nil
	}

	// Create the histogram. A zero-value config asks for 0 bins, which
	// plotter rejects; pick a usable default rather than failing on it.
	bins := config.Bins
	if bins <= 0 {
		bins = defaultHistogramBins
	}
	hist, err := plotter.NewHist(plotter.Values(values), bins)
	if err != nil {
		insyra.LogError("gplot", "CreateHistogram", "failed to build the histogram: %v", err)
		return nil
	}

	// Add the histogram to the plot.
	plt.Add(hist)

	return plt
}
