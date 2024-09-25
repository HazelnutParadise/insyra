// plot/radar.go

package plot

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// RadarChartConfig 定義雷達圖的配置
type RadarChartConfig struct {
	Title      string
	Subtitle   string
	Indicators []string // Optional: Automatically generated if not provided.
	MaxValues  map[string]float32
	SeriesData map[string]map[string]float32
}

// CreateRadarChart 生成並返回 *charts.Radar 對象
func CreateRadarChart(config RadarChartConfig) *charts.Radar {
	radar := charts.NewRadar()

	// 設置標題和副標題
	radar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Bottom: "0%",
		}),
	)

	if config.Indicators == nil {
		indicatorSet := make(map[string]struct{})

		// 從所有城市中提取所有指標
		for _, indicators := range config.SeriesData {
			for key := range indicators {
				indicatorSet[key] = struct{}{}
			}
		}

		// 將 map 中的 keys 轉換為 slice
		config.Indicators = make([]string, 0, len(indicatorSet))
		for key := range indicatorSet {
			config.Indicators = append(config.Indicators, key)
		}
	}

	// 計算缺失的最大值
	if config.MaxValues == nil {
		config.MaxValues = make(map[string]float32)
	}
	for _, indicator := range config.Indicators {
		if _, exists := config.MaxValues[indicator]; !exists {
			config.MaxValues[indicator] = calculateMaxValue(config.SeriesData, indicator)
		}
	}

	// 生成指標信息
	indicators := make([]*opts.Indicator, 0)
	for _, indicator := range config.Indicators {
		indicators = append(indicators, &opts.Indicator{
			Name: indicator,
			Max:  config.MaxValues[indicator],
		})
	}

	// 設置雷達圖的指標
	radar.SetGlobalOptions(
		charts.WithRadarComponentOpts(opts.RadarComponent{
			Indicator: indicators,
		}),
	)

	for city, indicatorData := range config.SeriesData {
		values := make([]float32, len(config.Indicators))
		for i, indicator := range config.Indicators {
			value, ok := indicatorData[indicator]
			if ok {
				values[i] = value
			} else {
				values[i] = 0 // 默認值
			}
		}

		// 使用指定顏色創建數據系列
		radar.AddSeries(city, []opts.RadarData{{Value: values, Name: city}})
	}

	return radar
}

// calculateMaxValue 計算指標的最大值
func calculateMaxValue(seriesData map[string]map[string]float32, indicator string) float32 {
	var maxValue float32
	for _, data := range seriesData {
		if value, ok := data[indicator]; ok && value > maxValue {
			maxValue = value
		}
	}
	// 增加一些冗餘，防止最大值過於貼近數據
	return maxValue * 1.1
}
