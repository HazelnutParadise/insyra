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

// defaultHistogramBins is used when HistogramConfig.Bins is left at its zero
// value, so a zero-value config still produces a chart.
const defaultHistogramBins = 10

// CreateHistogram draws the distribution of the values in data. A plain slice
// is passed as insyra.NewDataList(values).
//
// The values are read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is counted as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is counted as 0.
//
// It returns a nil chart and an error when data is nil or empty, or holds a
// NaN or an infinity, which no bin can hold.
func CreateHistogram(config HistogramConfig, data insyra.IDataList) (*plot.Plot, error) {
	if isNilList(data) {
		return nil, chartError("CreateHistogram", "no data to draw")
	}
	values := readValues(data)
	if len(values) == 0 {
		return nil, chartError("CreateHistogram", "the data list is empty")
	}
	// gonum bins the values without checking them: a NaN panicked on amd64 and
	// drew a wrong chart on arm64.
	if i := firstNonFinite(values); i >= 0 {
		return nil, chartError("CreateHistogram", "the value at index %d is %v, which no bin can hold", i, values[i])
	}

	// Create the histogram. A zero-value config asks for 0 bins, which
	// plotter rejects; pick a usable default rather than failing on it.
	bins := config.Bins
	if bins <= 0 {
		bins = defaultHistogramBins
	}
	hist, err := plotter.NewHist(plotter.Values(values), bins)
	if err != nil {
		return nil, chartError("CreateHistogram", "cannot draw the histogram: %w", err)
	}

	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Add the histogram to the plot.
	plt.Add(hist)

	return plt, nil
}
