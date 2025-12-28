package plot

import (
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

// ScatterPoint represents a 2D point for scatter charts.
type ScatterPoint struct {
	X float64
	Y float64
}

// ScatterChartConfig defines the configuration for a scatter chart.
type ScatterChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	XAxisName        string   // Optional: X-axis name.
	XAxisMin         *float64 // Optional: minimum value of X axis.
	XAxisMax         *float64 // Optional: maximum value of X axis.
	XAxisSplitNumber *int     // Optional: split number for X axis.
	XAxisFormatter   string   // Optional: label formatter for X axis, e.g. "{value}°C".

	YAxisName        string        // Optional: Y-axis name.
	YAxisMin         *float64      // Optional: minimum value of Y axis.
	YAxisMax         *float64      // Optional: maximum value of Y axis.
	YAxisSplitNumber *int          // Optional: split number for Y axis.
	YAxisFormatter   string        // Optional: label formatter for Y axis, e.g. "{value}°C".
	Colors           []string      // Optional: Colors for the scatter points.
	ShowLabels       bool          // Optional: Show labels on the scatter points.
	LabelPos         LabelPosition // Optional: Position of the labels, default is "right".
	SplitLine        bool          // Optional: Whether to show split lines on the X and Y axes.
	Symbol           []string      // Optional: Symbol of the scatter points. Default is "circle". If there are multiple series, you can specify different symbols for each series. If the length of the array is less than the number of series, the remaining series will repeat the order of the symbols.
	SymbolSize       int           // Optional: Size of the scatter points. Default is 10.
}

// CreateScatterChart generates and returns a *charts.Scatter object based on ScatterChartConfig.
func CreateScatterChart(config ScatterChartConfig, data map[string][]ScatterPoint) *charts.Scatter {
	scatter := charts.NewScatter()

	internal.SetBaseChartGlobalOptions(scatter, internal.BaseChartConfig{
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

	if config.Symbol == nil {
		config.Symbol = []string{"circle"}
	}
	if config.SymbolSize == 0 {
		config.SymbolSize = 10
	}

	// Build X and Y axis options with optional min/max/split/formatter and splitline
	xAxis := opts.XAxis{
		Name: config.XAxisName,
		SplitLine: &opts.SplitLine{
			Show: opts.Bool(config.SplitLine),
		},
	}
	if config.XAxisMin != nil {
		xAxis.Min = *config.XAxisMin
	}
	if config.XAxisMax != nil {
		xAxis.Max = *config.XAxisMax
	}
	if config.XAxisSplitNumber != nil {
		xAxis.SplitNumber = *config.XAxisSplitNumber
	}
	if config.XAxisFormatter != "" {
		xAxis.AxisLabel = &opts.AxisLabel{Formatter: types.FuncStr(config.XAxisFormatter)}
	}

	yAxis := opts.YAxis{
		Name: config.YAxisName,
		SplitLine: &opts.SplitLine{
			Show: opts.Bool(config.SplitLine),
		},
	}
	if config.YAxisMin != nil {
		yAxis.Min = *config.YAxisMin
	}
	if config.YAxisMax != nil {
		yAxis.Max = *config.YAxisMax
	}
	if config.YAxisSplitNumber != nil {
		yAxis.SplitNumber = *config.YAxisSplitNumber
	}
	if config.YAxisFormatter != "" {
		yAxis.AxisLabel = &opts.AxisLabel{Formatter: types.FuncStr(config.YAxisFormatter)}
	}

	// Apply axis options
	scatter.SetGlobalOptions(
		charts.WithXAxisOpts(xAxis),
		charts.WithYAxisOpts(yAxis),
	)

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		scatter.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 處理多個系列的點資料
	symbolIndex := 0
	for seriesName, pts := range data {
		if symbolIndex >= len(config.Symbol) {
			symbolIndex = 0
		}
		scatterData := convertToScatterData(pts, config, symbolIndex)
		symbolIndex++
		// 添加系列數據
		scatter.AddSeries(seriesName, scatterData)
	}

	// 設置標籤
	internal.SetShowLabels(scatter, config.ShowLabels, string(config.LabelPos), string(LabelPositionRight))

	return scatter
}

// convertToScatterData 將 []Point 轉換為 []opts.ScatterData
func convertToScatterData(data []ScatterPoint, config ScatterChartConfig, symbolIndex int) []opts.ScatterData {
	scatterData := make([]opts.ScatterData, len(data))
	for i, p := range data {
		scatterData[i] = opts.ScatterData{
			Value:      [2]float64{p.X, p.Y}, // 第一個是 X 值，第二個是 Y 值
			Symbol:     config.Symbol[symbolIndex],
			SymbolSize: config.SymbolSize,
		}
	}
	return scatterData
}
