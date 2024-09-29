// plot/gauge.go

package plot

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// GaugeChartConfig defines the configuration for a gauge chart.
type GaugeChartConfig struct {
	Title      string  // chart title
	Subtitle   string  // chart subtitle
	SeriesName string  // series name
	Value      float64 // value to display
}

// CreateGaugeChart 生成並返回 *charts.Gauge 對象
func CreateGaugeChart(config GaugeChartConfig) *charts.Gauge {
	gauge := charts.NewGauge()

	// 設置標題和副標題
	gauge.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
	)

	// 添加系列數據
	gauge.AddSeries(config.SeriesName, []opts.GaugeData{
		{Name: config.SeriesName, Value: config.Value},
	})

	return gauge
}
