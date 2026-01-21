// gplot/scatter.go

package gplot

import (
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

// CreateScatterPlot generates and returns a plot.Plot object based on ScatterPlotConfig.
// The Data field can be of type map[string][]float64, []*insyra.DataList, or []insyra.IDataList.
func CreateScatterPlot(config ScatterPlotConfig, data any) *plot.Plot {
	// Create a new plot.
	plt := plot.New()

	// Set chart title and axis labels.
	plt.Title.Text = config.Title
	plt.X.Label.Text = config.XAxisName
	plt.Y.Label.Text = config.YAxisName

	// Handle different types of Data
	switch data := data.(type) {
	case map[string][][]float64:
		// If Data is map[string][][]float64
		i := 0
		for seriesName, xyPairs := range data {
			addScatterSeries(plt, seriesName, xyPairs, i)
			i++
		}
	case []*insyra.DataList:
		// If Data is []*insyra.DataList
		for i, dataList := range data {
			xyPairs := convertDataListToXYPairs(dataList)
			addScatterSeries(plt, dataList.GetName(), xyPairs, i)
		}

	case []insyra.IDataList:
		// If Data is []insyra.IDataList
		for i, dataList := range data {
			xyPairs := convertIDataListToXYPairs(dataList)
			addScatterSeries(plt, dataList.GetName(), xyPairs, i)
		}
	default:
		insyra.LogWarning("gplot", "CreateScatterPlot", "Unsupported Data type: %T\n", data)
		return nil
	}

	return plt
}

// convertDataListToXYPairs converts a DataList to [][]float64 (X, Y pairs).
// Assumes the DataList contains alternating X and Y values or pairs of values.
func convertDataListToXYPairs(dataList *insyra.DataList) [][]float64 {
	values := dataList.ToF64Slice()
	if len(values)%2 != 0 {
		insyra.LogWarning("gplot", "convertDataListToXYPairs", "DataList %s has odd number of values, last value will be ignored", dataList.GetName())
		values = values[:len(values)-1]
	}

	xyPairs := make([][]float64, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		xyPairs[i/2] = []float64{values[i], values[i+1]}
	}
	return xyPairs
}

// convertIDataListToXYPairs converts an IDataList to [][]float64 (X, Y pairs).
func convertIDataListToXYPairs(dataList insyra.IDataList) [][]float64 {
	values := dataList.ToF64Slice()
	if len(values)%2 != 0 {
		insyra.LogWarning("gplot", "convertIDataListToXYPairs", "DataList %s has odd number of values, last value will be ignored", dataList.GetName())
		values = values[:len(values)-1]
	}

	xyPairs := make([][]float64, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		xyPairs[i/2] = []float64{values[i], values[i+1]}
	}
	return xyPairs
}

// addScatterSeries is a helper function to add a scatter series to the plot.
func addScatterSeries(plt *plot.Plot, seriesName string, xyPairs [][]float64, index int) {
	// Convert xyPairs to plotter.XYs
	scatterData := make(plotter.XYs, len(xyPairs))
	for i, pair := range xyPairs {
		if len(pair) != 2 {
			insyra.LogWarning("gplot", "addScatterSeries", "Invalid data point at index %d for series %s: expected [X, Y], got %v", i, seriesName, pair)
			continue
		}
		scatterData[i].X = pair[0]
		scatterData[i].Y = pair[1]
	}

	// Create the scatter plot
	scatter, err := plotter.NewScatter(scatterData)
	if err != nil {
		insyra.LogWarning("gplot", "addScatterSeries", "Failed to create scatter plot for series %s: %v", seriesName, err)
		return
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
}
