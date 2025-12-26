// plot/funnel.go

package plot

import (
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// FunnelChartConfig defines the configuration for a funnel chart.
type FunnelChartConfig struct {
	Width           string // Width of the chart (default "900px").
	Height          string // Height of the chart (default "500px").
	BackgroundColor string // Background color of the chart (default "white").
	Theme           Theme  // Theme of the chart.
	Title           string
	Subtitle        string
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	ShowLabels bool          // whether to show labels
	LabelPos   LabelPosition // Optional: Use const LabelPositionXXX.
}

// CreateFunnelChart generates a funnel chart based on the provided configuration.
// The data parameter is a map where keys are category names and values are their corresponding values.
func CreateFunnelChart(config FunnelChartConfig, data map[string]float64) *charts.Funnel {
	funnel := charts.NewFunnel()

	internal.SetBaseChartGlobalOptions(funnel, internal.BaseChartConfig{
		Width:           config.Width,
		Height:          config.Height,
		BackgroundColor: config.BackgroundColor,
		Theme:           string(config.Theme),
		Title:           config.Title,
		Subtitle:        config.Subtitle,
		TitlePos:        string(config.TitlePos),
		HideLegend:      config.HideLegend,
		LegendPos:       string(config.LegendPos),
	})

	// 構建數據點
	funnelData := make([]opts.FunnelData, 0)
	for name, value := range data {
		funnelData = append(funnelData, opts.FunnelData{
			Name:  name,
			Value: value,
		})
	}

	// 添加數據系列
	series := funnel.AddSeries("", funnelData)

	// 設置標籤顯示選項
	series.SetSeriesOptions(charts.WithLabelOpts(
		opts.Label{
			Show:     opts.Bool(config.ShowLabels),
			Position: string(config.LabelPos), // 標籤位置
		},
	))

	return funnel
}
