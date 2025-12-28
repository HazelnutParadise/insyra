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
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	XAxis     []string // X-axis data.
	XAxisName string   // Optional: X-axis name.

	// Y axis customization
	YAxisName        string   // Optional: Y-axis name.
	YAxis            []string // Optional: Y-axis category labels (if provided treated as category).
	YAxisMin         *float64 // Optional: minimum value of Y axis.
	YAxisMax         *float64 // Optional: maximum value of Y axis.
	YAxisSplitNumber *int     // Optional: split number for Y axis.
	YAxisFormatter   string   // Optional: label formatter for Y axis, e.g. "{value}°C".

	Colors     []string // Optional: Colors for the lines, for example: ["blue", "red"].
	ShowLabels bool     // Optional: Show labels on the lines.
	LabelPos   string   // Optional: "top" | "bottom" | "left" | "right", default: "top".
	Smooth     bool     // Optional: Make the lines smooth.
	FillArea   bool     // Optional: Fill the area under the lines.
}

// CreateLineChart generates and returns a *charts.Line object based on LineChartConfig.
func CreateLineChart(config LineChartConfig, data ...insyra.IDataList) *charts.Line {
	if len(data) == 0 {
		insyra.LogWarning("plot", "CreateLineChart", "No data available for line chart. Returning nil.")
		return nil
	}
	line := charts.NewLine()

	internal.SetBaseChartGlobalOptions(line, internal.BaseChartConfig{
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

	line.SetGlobalOptions(
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Bottom: "0%",
		}),
	)

	// 設置 X 軸名稱（如果提供）
	if config.XAxisName != "" {
		line.SetGlobalOptions(
			charts.WithXAxisOpts(opts.XAxis{
				Name: config.XAxisName,
			}),
		)
	}

	// Use internal helper to build Y axis and get converters; it updates config.YAxis if derived
	_, _, toLineData, _ := internal.ApplyYAxis(line, config.YAxisName, &config.YAxis, config.YAxisMin, config.YAxisMax, config.YAxisSplitNumber, config.YAxisFormatter, data...)

	// Y axis already applied by internal.ApplyYAxis; nothing more to do here.

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		line.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

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
			line.AddSeries(dl.GetName(), toLineData(dl))
		})
	}

	// 顯示標籤
	internal.SetShowLabels(line, config.ShowLabels, config.LabelPos, string(LabelPositionTop))

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
