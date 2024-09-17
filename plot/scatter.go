package plot

import (
	"fmt"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// ScatterChartConfig defines the configuration for a scatter chart.
type ScatterChartConfig struct {
	Title      string                 // Title of the chart.
	Subtitle   string                 // Subtitle of the chart.
	SeriesData map[string][][]float64 // Accepts multiple series, where each series is identified by a string and contains two-dimensional data (X, Y).
	XAxisName  string                 // Optional: X-axis name.
	YAxisName  string                 // Optional: Y-axis name.
	Colors     []string               // Optional: Colors for the scatter points.
	ShowLabels bool                   // Optional: Show labels on the scatter points.
	LabelPos   string                 // Optional: Position of the labels, default is "right".
	GridTop    string                 // Optional: Space between the top of the chart and the title. Default is "80".
	SplitLine  bool                   // Optional: Whether to show split lines on the X and Y axes.
	Symbol     string                 // Optional: Symbol of the scatter points. Default is "circle".
	SymbolSize int                    // Optional: Size of the scatter points. Default is 4.
}

// CreateScatterChart generates and returns a *charts.Scatter object based on ScatterChartConfig.
func CreateScatterChart(config ScatterChartConfig) *charts.Scatter {
	scatter := charts.NewScatter()

	if config.GridTop == "" {
		config.GridTop = "80"
	}
	if config.Symbol == "" {
		config.Symbol = "circle"
	}
	if config.SymbolSize == 0 {
		config.SymbolSize = 4
	}

	scatter.SetGlobalOptions(
		charts.WithLegendOpts(
			opts.Legend{
				Bottom: "0%",
			}),
		charts.WithGridOpts(opts.Grid{
			Top: config.GridTop,
		}),
	)

	// Set global options such as title and axis names
	scatter.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
	)

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		scatter.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 處理多個系列的二維數據
	for seriesName, seriesData := range config.SeriesData {
		scatterData := convertToScatterData(seriesData, config)

		// 添加系列數據
		scatter.AddSeries(seriesName, scatterData)
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
			charts.WithXAxisOpts(opts.XAxis{
				SplitLine: &opts.SplitLine{
					Show: opts.Bool(true),
				},
			}),
			charts.WithYAxisOpts(opts.YAxis{
				SplitLine: &opts.SplitLine{
					Show: opts.Bool(true),
				},
			}),
		)
	} else {
		scatter.SetGlobalOptions(
			charts.WithXAxisOpts(opts.XAxis{
				SplitLine: &opts.SplitLine{
					Show: opts.Bool(false),
				},
			}),
		)
	}

	return scatter
}

// convertToScatterData 將 [][]float64 轉換為 []opts.ScatterData
func convertToScatterData(data [][]float64, config ScatterChartConfig) []opts.ScatterData {
	scatterData := make([]opts.ScatterData, len(data))
	for i, v := range data {
		if len(v) != 2 {
			fmt.Printf("Invalid data at index %d: expected [X, Y], got %v\n", i, v)
			continue
		}
		scatterData[i] = opts.ScatterData{
			Value:      [2]float64{v[0], v[1]}, // 第一個是 X 值，第二個是 Y 值
			Symbol:     config.Symbol,
			SymbolSize: config.SymbolSize,
		}
	}
	return scatterData
}
