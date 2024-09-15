// plot/create_bar.go
package plot

import (
	"fmt"
	"io"
	"os"

	"github.com/HazelnutParadise/insyra" // 確保這是正確的導入路徑
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// BarChartConfig 定義柱狀圖的配置參數
type BarChartConfig struct {
	Title      string
	Subtitle   string
	XAxis      []string
	SeriesData any    // 可接受 map[string][]float64 或 []*insyra.DataList
	XAxisName  string // 可選
	YAxisName  string // 可選
	Colors     []string
	ShowLabels bool
	LabelPos   string
	OutputPath string // 可選，如果提供，將自動渲染並保存圖表
}

// CreateBarChart 根據 BarChartConfig 生成並返回一個 *charts.Bar 對象
func CreateBarChart(config BarChartConfig) (*charts.Bar, error) {
	bar := charts.NewBar()

	// 設置標題和副標題
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

	// 設置 Y 軸名稱（如果提供）
	if config.YAxisName != "" {
		bar.SetGlobalOptions(
			charts.WithYAxisOpts(opts.YAxis{
				Name: config.YAxisName,
			}),
		)
	}

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		bar.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 設置 X 軸標籤
	bar.SetXAxis(config.XAxis)

	// 添加系列數據，根據 SeriesData 的類型進行處理
	switch data := config.SeriesData.(type) {
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
		return nil, fmt.Errorf("unsupported SeriesData type: %T", config.SeriesData)
	}

	// 顯示標籤（如果啟用）
	if config.ShowLabels {
		bar.SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:     opts.Bool(true),
				Position: config.LabelPos,
			}),
		)
	}

	// 如果指定了輸出路徑，則渲染並保存圖表
	if config.OutputPath != "" {
		f, err := os.Create(config.OutputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s: %w", config.OutputPath, err)
		}
		defer f.Close()

		// 渲染圖表到指定文件
		if err := bar.Render(io.MultiWriter(f)); err != nil {
			return nil, fmt.Errorf("failed to render chart: %w", err)
		}
	}

	return bar, nil
}

// convertToBarDataFloat 將 []float64 轉換為 []opts.BarData
func convertToBarDataFloat(data []float64) []opts.BarData {
	barData := make([]opts.BarData, len(data))
	for i, v := range data {
		barData[i] = opts.BarData{Value: v}
	}
	return barData
}
