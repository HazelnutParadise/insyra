// plot/themeriver.go

package plot

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// ThemeRiverData define single data struct.
type ThemeRiverData struct {
	Date  string  // date, format: "yyyy/MM/dd"
	Value float64 // value
	Name  string  // name/series name
}

// ThemeRiverChartConfig define chart config, fixed X-axis to time axis.
type ThemeRiverChartConfig struct {
	Title    string           // title
	Subtitle string           // subtitle
	Data     []ThemeRiverData // data
}

// CreateThemeRiverChart create and return *charts.ThemeRiver object
func CreateThemeRiverChart(config ThemeRiverChartConfig) *charts.ThemeRiver {
	themeRiver := charts.NewThemeRiver()

	// 設置標題、軸類型和其他屬性，固定 X 軸為時間軸
	themeRiver.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithSingleAxisOpts(opts.SingleAxis{
			Type:   "time", // 固定為時間軸，使用斜線日期
			Bottom: "12%",  // 底部距離
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis", // 觸發方式為軸觸發
		}),
		charts.WithLegendOpts(opts.Legend{
			Bottom: "0%",
		}),
	)

	// 將 ThemeRiverData 數據轉換為 opts.ThemeRiverData
	convertedData := convertToThemeRiverData(config.Data)

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
