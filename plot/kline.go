// plot/kline.go

package plot

import (
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// KlineChartConfig defines the configuration for a K-line chart.
type KlineChartConfig struct {
	Title      string
	Subtitle   string
	SeriesData map[string][4]float32 // date: [open, close, lowest, highest]
	DataZoom   bool                  // Turn on/off data zoom
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

	// Prepare the K-line chart data with sorted dates
	xAxis := make([]string, 0, len(config.SeriesData))
	for date := range config.SeriesData {
		xAxis = append(xAxis, date)
	}

	// Sort the dates
	sort.Strings(xAxis)

	// Add sorted data to the series
	series := make([]opts.KlineData, len(xAxis))
	for i, date := range xAxis {
		series[i] = opts.KlineData{Value: config.SeriesData[date]}
	}

	// Set X axis and add series data
	kline.SetXAxis(xAxis).AddSeries("Kline", series)

	return kline
}
