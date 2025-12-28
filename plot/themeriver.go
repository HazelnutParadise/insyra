// plot/themeriver.go

package plot

import (
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type ThemeRiverAxisType string

// Other axis types (value, category, log) seem not inplemented in go-echarts ThemeRiver yet.
const (
	// ThemeRiverAxisTypeValue    ThemeRiverAxisType = "value"
	// ThemeRiverAxisTypeCategory ThemeRiverAxisType = "category"

	ThemeRiverAxisTypeTime ThemeRiverAxisType = "time"

	// ThemeRiverAxisTypeLog      ThemeRiverAxisType = "log"
)

// ThemeRiverData define single data struct.
type ThemeRiverData struct {
	Date  string  // date, format: "yyyy/MM/dd"
	Value float64 // value
	Name  string  // name/series name
}

// ThemeRiverChartConfig define chart config and X axis options.
type ThemeRiverChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	// Axis configuration: supports "value", "category", "time", "log".
	// If empty, defaults to "time" (preserves previous behavior).
	AxisType ThemeRiverAxisType // Use ThemeRiverAxisTypeXXX
	AxisData []string           // Optional: categories for category axis (xAxis.data)
	AxisMin  *float64           // Optional: min value for value/log axes
	AxisMax  *float64           // Optional: max value for value/log axes
}

// CreateThemeRiverChart create and return *charts.ThemeRiver object
func CreateThemeRiverChart(config ThemeRiverChartConfig, data ...ThemeRiverData) *charts.ThemeRiver {
	themeRiver := charts.NewThemeRiver()

	internal.SetBaseChartGlobalOptions(themeRiver, internal.BaseChartConfig{
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

	// 設置標題、軸類型和其他屬性。支援 axis types: value, category, time, log
	axisType := config.AxisType
	if axisType == "" {
		axisType = "time"
	}

	singleAxis := opts.SingleAxis{
		Type:   string(axisType),
		Bottom: "12%",
	}

	// 如果是 category 並提供 AxisData，category 值通常會從 series.data 或 dataset.source 自動取得。
	// go-echarts 的 SingleAxis 在此版本沒有提供 Data 欄位，因此不直接設定 singleAxis.Data。
	// 呼叫方可以把 AxisData 用在 series.data 或其他地方以確保類別順序。

	// 設定 min/max（用於 value 或 log）
	if config.AxisMin != nil {
		singleAxis.Min = *config.AxisMin
	}
	if config.AxisMax != nil {
		singleAxis.Max = *config.AxisMax
	}

	themeRiver.SetGlobalOptions(
		charts.WithSingleAxisOpts(singleAxis),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis", // 觸發方式為軸觸發
		}),
	)

	// 將 ThemeRiverData 數據轉換為 opts.ThemeRiverData
	convertedData := convertToThemeRiverData(data)

	// 添加數據系列
	themeRiver.AddSeries("themeRiver", convertedData)

	return themeRiver
}

// convertToThemeRiverData 將 []ThemeRiverData 轉換為 []opts.ThemeRiverData 格式
func convertToThemeRiverData(data []ThemeRiverData) []opts.ThemeRiverData {
	items := make([]opts.ThemeRiverData, len(data))
	for i, d := range data {
		items[i] = opts.ThemeRiverData{
			Date:  d.Date,  // 日期必須是 "yyyy/MM/dd" 格式
			Value: d.Value, // 數值
			Name:  d.Name,  // 系列名稱
		}
	}
	return items
}
