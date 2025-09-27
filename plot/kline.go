// plot/kline.go

package plot

import (
	"sort"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// KlineChartConfig defines the configuration for a K-line chart.
type KlineChartConfig struct {
	Title    string
	Subtitle string
	Data     any  // Accepts map[string][4]float32 or []*insyra.DataList
	DataZoom bool // Turn on/off data zoom
}

// CreateKlineChart generates and returns a *charts.Kline object.
func CreateKlineChart(config KlineChartConfig) *charts.Kline {
	kline := charts.NewKLine()

	// Set title and subtitle
	kline.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			SplitNumber: 20,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Scale: opts.Bool(true),
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
		charts.WithGridOpts(opts.Grid{
			Top: "80",
		}),
	)

	// Enable data zoom if needed
	if config.DataZoom {
		kline.SetGlobalOptions(
			charts.WithDataZoomOpts(opts.DataZoom{
				Start:      50,
				End:        100,
				XAxisIndex: []int{0},
			}),
		)
	}

	var xAxis []string
	var series []opts.KlineData

	// Handle Data for both map[string][4]float32 and []*insyra.DataList
	switch data := config.Data.(type) {
	case map[string][4]float32:
		// Prepare the K-line chart data with sorted dates
		xAxis = make([]string, 0, len(data))
		for date := range data {
			xAxis = append(xAxis, date)
		}
		sort.Strings(xAxis)

		// Add sorted data to the series
		series = make([]opts.KlineData, len(xAxis))
		for i, date := range xAxis {
			series[i] = opts.KlineData{Value: data[date]}
		}

	case []*insyra.DataList:
		// Prepare the K-line chart data using DataList
		xAxis = make([]string, 0, len(data))
		series = make([]opts.KlineData, 0, len(data))

		for _, dataList := range data {
			dataList.AtomicDo(func(dl *insyra.DataList) {
				if dl.Len() == 4 { // Ensure that we have the open, close, lowest, highest values
					xAxis = append(xAxis, dl.GetName())
					values := dl.ToF64Slice()
					series = append(series, opts.KlineData{
						Value: [4]float32{
							float32(values[0]), // Open
							float32(values[1]), // Close
							float32(values[2]), // Lowest
							float32(values[3]), // Highest
						},
					})
				}
			})
		}

		// Sort the dates
		sort.Strings(xAxis)
	case []insyra.IDataList:
		// Prepare the K-line chart data using DataList
		xAxis = make([]string, 0, len(data))
		series = make([]opts.KlineData, 0, len(data))

		for _, dataList := range data {
			dataList.AtomicDo(func(dl *insyra.DataList) {
				if dl.Len() == 4 { // Ensure that we have the open, close, lowest, highest values
					xAxis = append(xAxis, dl.GetName())
					values := dl.ToF64Slice()
					series = append(series, opts.KlineData{
						Value: [4]float32{
							float32(values[0]), // Open
							float32(values[1]), // Close
							float32(values[2]), // Lowest
							float32(values[3]), // Highest
						},
					})
				}
			})
		}

		// Sort the dates
		sort.Strings(xAxis)
	default:
		// Log a warning or handle unsupported data type
		return nil
	}

	// Set X axis and add series data
	kline.SetXAxis(xAxis).AddSeries("Kline", series)

	return kline
}
