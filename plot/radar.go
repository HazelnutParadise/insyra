// plot/radar.go

package plot

import (
	"sort"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// RadarChartConfig 定義雷達圖的配置
type RadarChartConfig struct {
	Width           string   // Width of the chart (default "900px").
	Height          string   // Height of the chart (default "500px").
	BackgroundColor string   // Background color of the chart (default "white").
	Theme           Theme    // Theme of the chart.
	Title           string   // Title of the chart.
	Subtitle        string   // Subtitle of the chart.
	TitlePos        Position // Optional: Use const PositionXXX.
	HideLegend      bool     // Optional: Whether to hide the legend.
	LegendPos       Position // Optional: Use const PositionXXX.

	Indicators []string           // Optional: Automatically generated if not provided.
	MaxValues  map[string]float32 // Optional: Automatically generated if not provided.
}

// RadarSeries 是單一系列資料
type RadarSeries struct {
	Name   string
	Values []float32 // 與 RadarDataset.Indicators 順序對齊
	Color  string    // optional
}

// CreateRadarChart 使用 `RadarChartConfig`（包含 `Indicators` 與 `MaxValues`）及一或多個 `RadarSeries` 生成並返回 *charts.Radar 對象
func CreateRadarChart(config RadarChartConfig, series []RadarSeries) *charts.Radar {
	radar := charts.NewRadar()

	internal.SetBaseChartGlobalOptions(radar, internal.BaseChartConfig{
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

	// 指標順序來源：優先使用 config.Indicators；若未提供，則從 config.MaxValues 的 key 推斷
	indicators := config.Indicators
	if len(indicators) == 0 {
		if len(config.MaxValues) > 0 {
			indicators = make([]string, 0, len(config.MaxValues))
			for k := range config.MaxValues {
				indicators = append(indicators, k)
			}
			sort.Strings(indicators) // 保證穩定順序
		} else {
			insyra.LogFatal("plot", "CreateRadarChart", "Indicators must be provided in RadarChartConfig when passing series directly")
		}
	}

	// 準備 MaxValues
	if config.MaxValues == nil {
		config.MaxValues = make(map[string]float32)
	}
	for i, indicator := range indicators {
		if _, exists := config.MaxValues[indicator]; !exists {
			config.MaxValues[indicator] = calculateMaxValueFromSeries(series, i)
		}
	}

	// 檢查 MaxValues 是否包含不在 Indicators 中的 key，並記錄警告
	indicatorSet := make(map[string]struct{}, len(indicators))
	for _, ind := range indicators {
		indicatorSet[ind] = struct{}{}
	}
	for k := range config.MaxValues {
		if _, ok := indicatorSet[k]; !ok {
			insyra.LogWarning("plot", "CreateRadarChart", "MaxValues contains key %s not present in Indicators; ignoring", k)
		}
	}

	// 生成指標信息
	optsIndicators := make([]*opts.Indicator, 0, len(indicators))
	for _, indicator := range indicators {
		optsIndicators = append(optsIndicators, &opts.Indicator{
			Name: indicator,
			Max:  config.MaxValues[indicator],
		})
	}

	// 設置雷達圖的指標
	radar.SetGlobalOptions(
		charts.WithRadarComponentOpts(opts.RadarComponent{
			Indicator: optsIndicators,
		}),
	)

	for _, s := range series {
		// 確保長度一致，若不足則補 0
		values := make([]float32, len(indicators))
		for i := range indicators {
			if i < len(s.Values) {
				values[i] = s.Values[i]
			} else {
				values[i] = 0
			}
		}
		radar.AddSeries(s.Name, []opts.RadarData{{Value: values, Name: s.Name}})
	}

	return radar
}

// calculateMaxValueFromSeries 計算指標（由 index 指定）在多個 RadarSeries 中的最大值，用於不使用 RadarDataset 的情況
func calculateMaxValueFromSeries(series []RadarSeries, idx int) float32 {
	if idx < 0 {
		return 0
	}
	var maxValue float32
	for _, s := range series {
		if idx < len(s.Values) && s.Values[idx] > maxValue {
			maxValue = s.Values[idx]
		}
	}
	return maxValue * 1.1
}
