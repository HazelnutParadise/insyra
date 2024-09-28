// plot/boxplot.go

package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// BoxPlotConfig defines the configuration for a box plot chart.
type BoxPlotConfig struct {
	Title      string
	Subtitle   string
	XAxis      []string // X-axis data.
	SeriesData any      // Accepts map[string][][]float64, []*insyra.DataList, or map[string]any for multiple series.
	GridTop    string   // Optional: Top grid line. Default: "80".
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

	// Process SeriesData for single or multiple series
	switch data := config.SeriesData.(type) {
	case map[string][][]float64:
		// Handle multiple series with [][]float64
		if config.XAxis == nil {
			config.XAxis = []string{}
			for i := 0; i < len(data); i++ {
				config.XAxis = append(config.XAxis, fmt.Sprintf("Category %d", i+1))
			}
		}
		for seriesName, seriesData := range data {
			boxPlotItems := generateBoxPlotItemsFromMapFloat64(seriesData)
			boxPlot.SetXAxis(config.XAxis).AddSeries(seriesName, boxPlotItems)
		}
	case map[string][]*insyra.DataList:
		// Handle multiple series with []*insyra.DataList
		if config.XAxis == nil {
			config.XAxis = []string{}
			for i := 0; i < len(data); i++ {
				config.XAxis = append(config.XAxis, fmt.Sprintf("Category %d", i+1))
			}
		}
		for seriesName, seriesData := range data {
			boxPlotItems := generateBoxPlotItemsFromDataList(seriesData)
			boxPlot.SetXAxis(config.XAxis).AddSeries(seriesName, boxPlotItems)
		}
	case map[string][]insyra.IDataList:
		// Handle multiple series with []insyra.IDataList
		if config.XAxis == nil {
			config.XAxis = []string{}
			for i := 0; i < len(data); i++ {
				config.XAxis = append(config.XAxis, fmt.Sprintf("Category %d", i+1))
			}
		}
		for seriesName, seriesData := range data {
			boxPlotItems := generateBoxPlotItemsFromIDataList(seriesData)
			boxPlot.SetXAxis(config.XAxis).AddSeries(seriesName, boxPlotItems)
		}
	default:
		insyra.LogWarning("plot.CreateBoxPlot: Unsupported SeriesData type")
		return nil
	}

	return boxPlot
}

// createBoxPlotData generates the five-number summary (Min, Q1, Q2, Q3, Max)
func createBoxPlotData(data []float64) []float64 {
	dl := insyra.NewDataList(data)
	return []float64{
		dl.Min(),
		dl.Quartile(1),
		dl.Quartile(2),
		dl.Quartile(3),
		dl.Max(),
	}
}

// generateBoxPlotItemsFromMapFloat64 processes [][]float64 and generates BoxPlotData
func generateBoxPlotItemsFromMapFloat64(data [][]float64) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, len(data))
	for i, d := range data {
		items[i] = opts.BoxPlotData{Value: createBoxPlotData(d)}
	}
	return items
}

// generateBoxPlotItemsFromDataList generates BoxPlotData from []*insyra.DataList
func generateBoxPlotItemsFromDataList(dataLists []*insyra.DataList) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, len(dataLists))
	for i, dataList := range dataLists {
		values := dataList.ToF64Slice() // Convert DataList to []float64
		items[i] = opts.BoxPlotData{Value: createBoxPlotData(values)}
	}
	return items
}

// generateBoxPlotItemsFromIDataList generates BoxPlotData from []insyra.IDataList
func generateBoxPlotItemsFromIDataList(dataLists []insyra.IDataList) []opts.BoxPlotData {
	items := make([]opts.BoxPlotData, len(dataLists))
	for i, dataList := range dataLists {
		values := dataList.ToF64Slice() // Convert IDataList to []float64
		items[i] = opts.BoxPlotData{Value: createBoxPlotData(values)}
	}
	return items
}
