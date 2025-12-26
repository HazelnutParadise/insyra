// plot/bar.go

package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// BarChartConfig defines the configuration for a bar chart.
type BarChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	XAxis      []string      // X-axis data.
	XAxisName  string        // Optional: X-axis name.
	YAxisName  string        // Optional: Y-axis name.
	Colors     []string      // Optional: Colors for the bars, for example: ["green", "orange"].
	ShowLabels bool          // Optional: Show labels on the bars.
	LabelPos   LabelPosition // Optional: Use const LabelPositionXXX.
}

// CreateBarChart generates and returns a *charts.Bar object based on BarChartConfig.
func CreateBarChart(config BarChartConfig, data ...insyra.IDataList) *charts.Bar {
	bar := charts.NewBar()

	// Set title and subtitle
	internal.SetBaseChartGlobalOptions(bar, internal.BaseChartConfig{
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

	// 設置 X 軸名稱（如果提供）
	bar.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
		charts.WithColorsOpts(opts.Colors(config.Colors)),
	)

	if len(config.XAxis) == 0 {
		// 如果 X 軸沒有提供，則根據數據長度生成默認標籤
		var maxDataLength int

		for _, dataList := range data {
			dataList.AtomicDo(func(dl *insyra.DataList) {
				maxDataLength = max(dl.Len(), maxDataLength)
			})
		}

		// 生成 1, 2, 3, ... n 的 X 軸標籤
		config.XAxis = make([]string, maxDataLength)
		for i := 0; i < maxDataLength; i++ {
			config.XAxis[i] = fmt.Sprintf("%d", i+1)
		}
	}

	// 設置 X 軸標籤
	bar.SetXAxis(config.XAxis)

	// 添加系列數據
	for _, dataList := range data {
		dataList.AtomicDo(func(dl *insyra.DataList) {
			bar.AddSeries(dl.GetName(), convertToBarDataFloat(dl.ToF64Slice()))
		})
	}

	// 顯示標籤（如果啟用）
	if config.ShowLabels {
		if config.LabelPos == "" {
			config.LabelPos = "top"
		}
	}
	bar.SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:     opts.Bool(config.ShowLabels),
			Position: string(config.LabelPos),
		}),
	)

	return bar
}

// convertToBarDataFloat 將 []float64 轉換為 []opts.BarData
func convertToBarDataFloat(data []float64) []opts.BarData {
	barData := make([]opts.BarData, len(data))
	for i, v := range data {
		barData[i] = opts.BarData{Value: v}
	}
	return barData
}
