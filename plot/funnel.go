// plot/funnel.go

package plot

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// FunnelChartConfig defines the configuration for a funnel chart.
type FunnelChartConfig struct {
	Title      string             // chart title
	Subtitle   string             // chart subtitle
	SeriesName string             // series name
	Data       map[string]float64 // data points (dimension name -> value)
	ShowLabels bool               // whether to show labels
	LabelPos   string             // label position (e.g., "left", "right")
}

// CreateFunnelChart generates a funnel chart based on the provided configuration.
func CreateFunnelChart(config FunnelChartConfig) *charts.Funnel {
	funnel := charts.NewFunnel()

	// 設置標題和副標題
	funnel.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Bottom: "0%",
		}),
		charts.WithGridOpts(opts.Grid{
			Top: "80%",
		}),
	)

	// 構建數據點
	funnelData := make([]opts.FunnelData, 0)
	for name, value := range config.Data {
		funnelData = append(funnelData, opts.FunnelData{
			Name:  name,
			Value: value,
		})
	}

	// 添加數據系列
	series := funnel.AddSeries(config.SeriesName, funnelData)

	// 設置標籤顯示選項
	series.SetSeriesOptions(charts.WithLabelOpts(
		opts.Label{
			Show:     opts.Bool(config.ShowLabels),
			Position: config.LabelPos, // 標籤位置
		},
	))

	return funnel
}
