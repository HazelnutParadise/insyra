// plot/scatter.go

package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra" // 確保這是正確的導入路徑
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// ScatterChartConfig defines the configuration for a scatter chart.
type ScatterChartConfig struct {
	Title        string   // Title of the chart.
	Subtitle     string   // Subtitle of the chart.
	XAxis        []string // X-axis data (categories).
	SeriesData   any      // Accepts map[string][]float64, []*insyra.DataList, or []insyra.IDataList.
	XAxisName    string   // Optional: X-axis name.
	YAxisName    string   // Optional: Y-axis name.
	Colors       []string // Optional: Colors for the scatter points.
	ShowLabels   bool     // Optional: Show labels on the scatter points.
	LabelPos     string   // Optional: Position of the labels, default is "right".
	GridTop      string   // Optional: default: "80".
	Symbol       string   // Optional: Symbol style, for example: "circle", "roundRect".
	SymbolSize   int      // Optional: Size of the symbol.
	SymbolRotate int      // Optional: Rotation of the symbol.
	SplitLine    bool     // Optional: Show split line.
}

// CreateScatterChart generates and returns a *charts.Scatter object based on ScatterChartConfig.
func CreateScatterChart(config ScatterChartConfig) *charts.Scatter {
	scatter := charts.NewScatter()

	// Set title and subtitle
	scatter.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
	)

	// 設置 X 軸名稱（如果提供）
	if config.XAxisName != "" {
		scatter.SetGlobalOptions(
			charts.WithXAxisOpts(opts.XAxis{
				Name: config.XAxisName,
			}),
		)
	}

	// 設置 Y 軸名稱（如果提供）
	if config.YAxisName != "" {
		scatter.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{
				Name: config.YAxisName,
			}),
		)
	}

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		scatter.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	if len(config.XAxis) == 0 {
		// 如果 X 軸沒有提供，則根據數據長度生成默認標籤
		var maxDataLength int
		switch data := config.SeriesData.(type) {
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
	scatter.SetXAxis(config.XAxis)

	// 添加系列數據，根據 SeriesData 的類型進行處理
	switch data := config.SeriesData.(type) {
	case map[string][]float64:
		for name, vals := range data {
			scatter.AddSeries(name, convertToScatterData(vals, config))
		}
	case []*insyra.DataList:
		for _, dataList := range data {
			scatter.AddSeries(dataList.GetName(), convertToScatterData(dataList.ToF64Slice(), config))
		}
	case []insyra.IDataList:
		for _, dataList := range data {
			scatter.AddSeries(dataList.GetName(), convertToScatterData(dataList.ToF64Slice(), config))
		}
	default:
		insyra.LogWarning("unsupported SeriesData type: %T", config.SeriesData)
		return nil
	}

	// 設置標籤和符號選項
	if config.ShowLabels {
		if config.LabelPos == "" {
			config.LabelPos = "right"
		}
		scatter.SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:     opts.Bool(true),
				Position: config.LabelPos,
			}),
		)
	}

	if config.SplitLine {
		scatter.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{
				SplitLine: &opts.SplitLine{
					Show: opts.Bool(true),
				},
			}),
			charts.WithXAxisOpts(opts.XAxis{
				SplitLine: &opts.SplitLine{
					Show: opts.Bool(true),
				},
			}),
		)
	}

	return scatter
}

// convertToScatterData 將 []float64 轉換為 []opts.ScatterData
func convertToScatterData(data []float64, config ScatterChartConfig) []opts.ScatterData {
	scatterData := make([]opts.ScatterData, len(data))
	for i, v := range data {
		scatterData[i] = opts.ScatterData{
			Value:        v,
			Symbol:       config.Symbol,
			SymbolSize:   config.SymbolSize,
			SymbolRotate: config.SymbolRotate,
		}
	}
	return scatterData
}
