package plot

import (
	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// PieItem represents a single item in a pie chart.
type PieItem struct {
	Name  string  // Label/name of the pie slice.
	Value float64 // Value of the pie slice.
}

// PieChartConfig defines the configuration for a pie chart.
type PieChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	Colors      []string // Optional: Colors for the slices, for example: ["green", "orange"].
	ShowLabels  bool     // Optional: Show labels on the slices.
	ShowPercent bool     // Optional: Show percentage on labels.
	LabelPos    LabelPosition
	RoseType    string   // Optional: "radius" or "area" for rose charts.
	Radius      []string // Optional: Radius configuration. First value is inner radius, second is outer radius, for example: ["40%", "75%"].
	Center      []string // Optional: Center position, for example: ["50%", "50%"].
}

// CreatePieChart generates and returns a *charts.Pie object based on PieChartConfig.
func CreatePieChart(config PieChartConfig, data ...PieItem) *charts.Pie {
	pie := charts.NewPie()

	internal.SetBaseChartGlobalOptions(pie, internal.BaseChartConfig{
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

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		pie.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 檢查數據是否為空
	if len(data) == 0 {
		insyra.LogWarning("plot", "CreatePieChart", "Data is empty")
		return nil
	}

	// 轉換數據為 pie 數據
	convertedData := convertToPieData(data)
	pie.AddSeries("pie", convertedData)

	// 設置標籤和其他選項
	labelFormatter := "{b}: {c}" // 預設顯示名稱和值
	if config.ShowPercent && config.ShowLabels {
		labelFormatter = "{b}: {c}\n({d}%)" // 增加百分比顯示
	} else if config.ShowPercent {
		labelFormatter = "{d}%"
	}

	if config.Center == nil {
		config.Center = []string{"50%", "50%"}
	}
	if config.Radius == nil {
		config.Radius = []string{"50%", "70%"}
	}

	pie.SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:      opts.Bool(config.ShowLabels || config.ShowPercent),
			Position:  string(config.LabelPos),
			Formatter: labelFormatter,
		}),
		charts.WithPieChartOpts(opts.PieChart{
			RoseType: config.RoseType,
			Radius:   config.Radius,
			Center:   config.Center,
		}),
	)

	return pie
}

// convertToPieData 將 []PieItem 轉換為 []opts.PieData
func convertToPieData(data []PieItem) []opts.PieData {
	pieData := make([]opts.PieData, 0, len(data))
	for _, item := range data {
		pieData = append(pieData, opts.PieData{Name: item.Name, Value: item.Value})
	}
	return pieData
}
