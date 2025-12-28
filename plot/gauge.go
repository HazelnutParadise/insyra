// plot/gauge.go

package plot

import (
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// GaugeChartConfig defines the configuration for a gauge chart.
type GaugeChartConfig struct {
	Width           string // Width of the chart (default "900px").
	Height          string // Height of the chart (default "500px").
	BackgroundColor string // Background color of the chart (default "white").
	Theme           Theme  // Theme of the chart.
	Title           string
	Subtitle        string
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	SeriesName string // series name
}

// CreateGaugeChart generates and returns a *charts.Gauge object
func CreateGaugeChart(config GaugeChartConfig, value float64) *charts.Gauge {
	gauge := charts.NewGauge()

	internal.SetBaseChartGlobalOptions(gauge, internal.BaseChartConfig{
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

	// 添加系列數據
	gauge.AddSeries(config.SeriesName, []opts.GaugeData{
		{Name: config.SeriesName, Value: value},
	})

	return gauge
}
