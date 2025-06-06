// plot/bar.go

package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// BarChartConfig defines the configuration for a bar chart.
type BarChartConfig struct {
	Title        string   // Title of the chart.
	Subtitle     string   // Subtitle of the chart.
	XAxis        []string // X-axis data.
	Data         any      // Accepts map[string][]float64, []*insyra.DataList, or []insyra.IDataList.
	XAxisName    string   // Optional: X-axis name.
	YAxisName    string   // Optional: Y-axis name.
	YAxisNameGap int      // Optional: Gap between Y-axis name and subtitle.
	Colors       []string // Optional: Colors for the bars, for example: ["green", "orange"].
	ShowLabels   bool     // Optional: Show labels on the bars.
	LabelPos     string   // Optional: "top" | "bottom" | "left" | "right", default: "top".
	GridTop      string   // Optional: default: "80".
}

// CreateBarChart generates and returns a *charts.Bar object based on BarChartConfig.
func CreateBarChart(config BarChartConfig) *charts.Bar {
	bar := charts.NewBar()

	bar.SetGlobalOptions(
		charts.WithLegendOpts(opts.Legend{
			Bottom: "0%",
		}),
	)

	// Set title and subtitle
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
	)

	// 設置 X 軸名稱（如果提供）
	if config.XAxisName != "" {
		bar.SetGlobalOptions(
			charts.WithXAxisOpts(opts.XAxis{
				Name: config.XAxisName,
			}),
		)
	}

	// 設置 Y 軸名稱和 NameGap（增加間距）
	if config.YAxisName != "" {
		bar.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{
				Name:    config.YAxisName,
				NameGap: config.YAxisNameGap, // 使用配置中的 NameGap
			}),
		)
	}

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		bar.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 設置 GridTop 以增加副標題下的空間
	if config.GridTop != "" {
		bar.SetGlobalOptions(
			charts.WithGridOpts(opts.Grid{
				Top: config.GridTop,
			}),
		)
	} else {
		bar.SetGlobalOptions(
			charts.WithGridOpts(opts.Grid{
				Top: "80",
			}),
		)
	}

	if len(config.XAxis) == 0 {
		// 如果 X 軸沒有提供，則根據數據長度生成默認標籤
		var maxDataLength int
		switch data := config.Data.(type) {
		case map[string][]float64:
			for _, vals := range data {
				if len(vals) > maxDataLength {
					maxDataLength = len(vals)
				}
			}
		case []*insyra.DataList:
			for _, dataList := range data {
				if dataList.Len() > maxDataLength {
					maxDataLength = len(dataList.ToF64Slice())
				}
			}
		case []insyra.IDataList:
			for _, dataList := range data {
				if dataList.Len() > maxDataLength {
					maxDataLength = len(dataList.ToF64Slice())
				}
			}
		}

		// 生成 1, 2, 3, ... n 的 X 軸標籤
		config.XAxis = make([]string, maxDataLength)
		for i := 0; i < maxDataLength; i++ {
			config.XAxis[i] = fmt.Sprintf("%d", i+1)
		}
	}

	// 設置 X 軸標籤
	bar.SetXAxis(config.XAxis)

	// 添加系列數據，根據 Data 的類型進行處理
	switch data := config.Data.(type) {
	case map[string][]float64:
		for name, vals := range data {
			bar.AddSeries(name, convertToBarDataFloat(vals))
		}
	case []*insyra.DataList:
		for _, dataList := range data {
			bar.AddSeries(dataList.GetName(), convertToBarDataFloat(dataList.ToF64Slice()))
		}
	case []insyra.IDataList:
		for _, dataList := range data {
			bar.AddSeries(dataList.GetName(), convertToBarDataFloat(dataList.ToF64Slice()))
		}
	default:
		insyra.LogWarning("plot", "CreateBarChart", "unsupported Data type: %T", config.Data)
		return nil
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
			Position: config.LabelPos,
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
