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
	SeriesData any    // Accepts map[string][]float64 or []*insyra.DataList.
	GridTop    string // Optional: Top grid line. Default: "80".
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

	var boxPlotItems []opts.BoxPlotData
	var xAxis []string

	// Handle SeriesData as map[string][]float64 or []*insyra.DataList
	switch data := config.SeriesData.(type) {
	case map[string][]float64:
		boxPlotItems, xAxis = generateBoxPlotItemsFromMapFloat64(data)
	case []*insyra.DataList:
		boxPlotItems, xAxis = generateBoxPlotItemsFromDataList(data)
	case []insyra.IDataList:
		boxPlotItems, xAxis = generateBoxPlotItemsFromIDataList(data)
	default:
		insyra.LogWarning("plot.CreateBoxPlot: Unsupported SeriesData type")
		return nil
	}

	// Set X-axis data if not provided, auto-generate from DataList
	if len(xAxis) > 0 {
		boxPlot.SetXAxis(xAxis).AddSeries("boxplot", boxPlotItems)
	} else {
		boxPlot.AddSeries("boxplot", boxPlotItems)
	}

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

// generateBoxPlotItemsFromMapFloat64 creates a list of opts.BoxPlotData for map[string][]float64 and auto-generates X-axis
func generateBoxPlotItemsFromMapFloat64(dataMap map[string][]float64) ([]opts.BoxPlotData, []string) {
	items := make([]opts.BoxPlotData, 0)
	xAxis := make([]string, len(dataMap))
	i := 0
	for name, boxPlotData := range dataMap {
		items = append(items, opts.BoxPlotData{Value: createBoxPlotData(boxPlotData)})
		xAxis[i] = name // Use map keys as X-axis labels
		i++
	}
	return items, xAxis
}

// generateBoxPlotItemsFromDataList creates a list of opts.BoxPlotData for []*insyra.DataList and auto-generates X-axis
func generateBoxPlotItemsFromDataList(dataLists []*insyra.DataList) ([]opts.BoxPlotData, []string) {
	items := make([]opts.BoxPlotData, 0)
	xAxis := make([]string, len(dataLists))
	for i, dataList := range dataLists {
		values := dataList.ToF64Slice() // Convert DataList to []float64
		items = append(items, opts.BoxPlotData{Value: createBoxPlotData(values)})
		xAxis[i] = dataList.GetName() // Use the DataList name as the X-axis label
	}
	return items, xAxis
}

// generateBoxPlotItemsFromIDataList creates a list of opts.BoxPlotData for []insyra.IDataList
func generateBoxPlotItemsFromIDataList(dataLists []insyra.IDataList) ([]opts.BoxPlotData, []string) {
	items := make([]opts.BoxPlotData, 0)
	xAxis := make([]string, len(dataLists))
	for i, dataList := range dataLists {
		values := dataList.ToF64Slice() // Convert IDataList to []float64
		items = append(items, opts.BoxPlotData{Value: createBoxPlotData(values)})
		xAxis[i] = dataList.GetName() // Use the IDataList name as the X-axis label
	}
	return items, xAxis
}
