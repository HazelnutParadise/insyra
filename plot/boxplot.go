// plot/boxplot.go

package plot

import (
	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// BoxPlotConfig defines the configuration for a box plot chart.
type BoxPlotConfig struct {
	Title      string
	Subtitle   string
	XAxis      []string               // X-axis data.
	SeriesData map[string]interface{} // Accepts map[string][][]float64 or map[string][]*insyra.DataList for multiple series.
	GridTop    string                 // Optional: Top grid line. Default: "80".
}

// CreateBoxPlot generates and returns a *charts.BoxPlot object
func CreateBoxPlot(config BoxPlotConfig) *charts.BoxPlot {
	boxPlot := charts.NewBoxPlot()

	if config.GridTop == "" {
		config.GridTop = "80"
	}

	// Set title and subtitle
	boxPlot.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Bottom: "bottom",
		}),
		charts.WithGridOpts(opts.Grid{
			Top: config.GridTop,
		}),
	)

	// Process SeriesData for multiple series
	for seriesName, data := range config.SeriesData {
		var boxPlotItems []opts.BoxPlotData

		switch d := data.(type) {
		case [][]float64:
			boxPlotItems = generateBoxPlotItems(d)
		case []*insyra.DataList:
			boxPlotItems = generateBoxPlotItemsFromDataList(d)
		case []insyra.IDataList:
			boxPlotItems = generateBoxPlotItemsFromIDataList(d)
		default:
			insyra.LogWarning("plot.CreateBoxPlot: Unsupported SeriesData type")
			continue
		}

		// Add each series to the box plot
		boxPlot.AddSeries(seriesName, boxPlotItems)
	}

	// Set X-axis data
	boxPlot.SetXAxis(config.XAxis)

	return boxPlot
}

// createBoxPlotData generates the five-number summary (Min, Q1, Q2, Q3, Max)
func createBoxPlotData(data []float64) []float64 {
	dl := insyra.NewDataList(data)
	min := dl.Min()
	max := dl.Max()
	q1 := dl.Quartile(1)
	q2 := dl.Quartile(2)
	q3 := dl.Quartile(3)

	return []float64{
		min,
		q1,
		q2,
		q3,
		max,
	}
}

// generateBoxPlotItems creates a list of opts.BoxPlotData for [][]float64
func generateBoxPlotItems(boxPlotData [][]float64) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, 0)
	for i := 0; i < len(boxPlotData); i++ {
		items = append(items, opts.BoxPlotData{Value: createBoxPlotData(boxPlotData[i])})
	}
	return items
}

// generateBoxPlotItemsFromIDataList creates a list of opts.BoxPlotData for []insyra.IDataList
func generateBoxPlotItemsFromIDataList(dataLists []insyra.IDataList) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, 0)
	for _, dataList := range dataLists {
		values := dataList.ToF64Slice() // Convert DataList to []float64
		items = append(items, opts.BoxPlotData{Value: createBoxPlotData(values)})
	}
	return items
}

// generateBoxPlotItemsFromDataList creates a list of opts.BoxPlotData for []*insyra.DataList
func generateBoxPlotItemsFromDataList(dataLists []*insyra.DataList) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, 0)
	for _, dataList := range dataLists {
		values := dataList.ToF64Slice() // Convert DataList to []float64
		items = append(items, opts.BoxPlotData{Value: createBoxPlotData(values)})
	}
	return items
}
