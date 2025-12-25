// plot/line.go

package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// LineChartConfig defines the configuration for a line chart.
type LineChartConfig struct {
	Title        string   // Title of the chart.
	Subtitle     string   // Subtitle of the chart.
	HideLegend   bool     // Optional: Whether to hide the legend.
	XAxis        []string // X-axis data.
	Data         []insyra.IDataList
	XAxisName    string   // Optional: X-axis name.
	YAxisName    string   // Optional: Y-axis name.
	YAxisNameGap int      // Optional: Gap between Y-axis name and subtitle.
	Colors       []string // Optional: Colors for the lines, for example: ["blue", "red"].
	ShowLabels   bool     // Optional: Show labels on the lines.
	LabelPos     string   // Optional: "top" | "bottom" | "left" | "right", default: "top".
	Smooth       bool     // Optional: Make the lines smooth.
	FillArea     bool     // Optional: Fill the area under the lines.
	GridTop      string   // Optional, default: "80".
}

// CreateLineChart generates and returns a *charts.Line object based on LineChartConfig.
func CreateLineChart(config LineChartConfig) *charts.Line {
	line := charts.NewLine()

	line.SetGlobalOptions(
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Bottom: "0%",
		}),
	)

	// Set title and subtitle
	internal.SetBaseChartGlobalOptions(&line.RectChart, internal.BaseChartConfig{
		Title:      config.Title,
		Subtitle:   config.Subtitle,
		HideLegend: config.HideLegend,
	})

	// 設置 X 軸名稱（如果提供）
	if config.XAxisName != "" {
		line.SetGlobalOptions(
			charts.WithXAxisOpts(opts.XAxis{
				Name: config.XAxisName,
			}),
		)
	}

	// 設置 Y 軸名稱和 NameGap（增加間距）
	if config.YAxisName != "" {
		line.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{
				Name:    config.YAxisName,
				NameGap: config.YAxisNameGap, // 使用配置中的 NameGap
			}),
		)
	}

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		line.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 設置 GridTop 以增加副標題下的空間
	if config.GridTop != "" {
		line.SetGlobalOptions(
			charts.WithGridOpts(opts.Grid{
				Top: config.GridTop,
			}),
		)
	} else {
		line.SetGlobalOptions(
			charts.WithGridOpts(opts.Grid{
				Top: "80",
			}),
		)
	}

	data := config.Data

	if len(config.XAxis) == 0 {
		// 如果 X 軸沒有提供，則根據數據長度生成默認標籤
		var maxDataLength int

		for _, dataList := range data {
			dataList.AtomicDo(func(dl *insyra.DataList) {
				if dl.Len() > maxDataLength {
					maxDataLength = len(dl.ToF64Slice())
				}
			})
		}

		// 生成 1, 2, 3, ... n 的 X 軸標籤
		config.XAxis = make([]string, maxDataLength)
		for i := 0; i < maxDataLength; i++ {
			config.XAxis[i] = fmt.Sprintf("%d", i+1)
		}
	}

	// 設置 X 軸標籤
	line.SetXAxis(config.XAxis)

	// 添加系列數據
	for _, dataList := range data {
		dataList.AtomicDo(func(dl *insyra.DataList) {
			line.AddSeries(dl.GetName(), convertToLineDataFloat(dl.ToF64Slice()))
		})
	}

	// 顯示標籤（如果啟用）
	if config.ShowLabels {
		if config.LabelPos == "" {
			config.LabelPos = "top"
		}
	}
	line.SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:     opts.Bool(config.ShowLabels),
			Position: config.LabelPos,
		}),
	)

	// 平滑線條（如果啟用）
	if config.Smooth {
		line.SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
		)
	}

	// 填充區域（如果啟用）
	if config.FillArea {
		line.SetSeriesOptions(
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: opts.Float(0.5), // 設置填充區域的透明度
			}),
		)
	}

	return line
}

// convertToLineDataFloat 將 []float64 轉換為 []opts.LineData
func convertToLineDataFloat(data []float64) []opts.LineData {
	lineData := make([]opts.LineData, len(data))
	for i, v := range data {
		lineData[i] = opts.LineData{Value: v}
	}
	return lineData
}
