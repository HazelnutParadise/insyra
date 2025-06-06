package plot

import (
	"errors"

	"github.com/HazelnutParadise/insyra" // 確保這是正確的導入路徑
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// PieChartConfig defines the configuration for a pie chart.
type PieChartConfig struct {
	Title       string   // Title of the chart.
	Subtitle    string   // Subtitle of the chart.
	Data        any      // Accepts []float64 or []*insyra.DataList.
	Labels      []string // Labels for each slice (for example: product names).
	Colors      []string // Optional: Colors for the slices, for example: ["green", "orange"].
	ShowLabels  bool     // Optional: Show labels on the slices.
	LabelPos    string   // Optional: "inside" | "outside", default: "outside".
	RoseType    string   // Optional: "radius" or "area" for rose charts.
	Radius      []string // Optional: Radius configuration. First value is inner radius, second is outer radius, for example: ["40%", "75%"].
	Center      []string // Optional: Center position, for example: ["50%", "50%"].
	ShowPercent bool     // Optional: Show percentage on labels.
}

// CreatePieChart generates and returns a *charts.Pie object based on PieChartConfig.
func CreatePieChart(config PieChartConfig) *charts.Pie {
	pie := charts.NewPie()

	pie.SetGlobalOptions(
		charts.WithLegendOpts(opts.Legend{
			Bottom: "0%",
		}),
	)

	// Set title and subtitle
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
	)

	// 設置系列顏色（如果提供）
	if len(config.Colors) > 0 {
		pie.SetGlobalOptions(
			charts.WithColorsOpts(opts.Colors(config.Colors)),
		)
	}

	// 添加系列數據，根據 Data 的類型進行處理
	switch data := config.Data.(type) {
	case []float64:
		if len(config.Labels) != len(data) {
			insyra.LogWarning("plot", "CreatePieChart", "The length of Labels and Data does not match.")
			return nil
		}
		convertedData, err := convertToPieData(data, config.Labels)
		if err != nil {
			insyra.LogWarning("plot", "CreatePieChart", "Failed to convert data: %v", err)
			return nil
		}
		pie.AddSeries("pie", convertedData)
	case *insyra.DataList:
		// 假設 DataList 提供了方法來返回數據
		values := data.ToF64Slice() // 返回 []float64 數據
		if len(config.Labels) != len(values) {
			insyra.LogWarning("plot", "CreatePieChart", "The length of Labels and DataList values does not match.")
			return nil
		}
		// 使用 convertToPieData 轉換數據
		convertedData, err := convertToPieData(values, config.Labels)
		if err != nil {
			insyra.LogWarning("plot", "CreatePieChart", "Failed to convert DataList values: %v", err)
			return nil
		}
		pie.AddSeries(data.GetName(), convertedData)
	case insyra.IDataList:
		values := data.ToF64Slice() // 返回 []float64 數據
		if len(config.Labels) != len(values) {
			insyra.LogWarning("plot", "CreatePieChart", "The length of Labels and DataList values does not match.")
			return nil
		}
		// 使用 convertToPieData 轉換數據
		convertedData, err := convertToPieData(values, config.Labels)
		if err != nil {
			insyra.LogWarning("plot", "CreatePieChart", "Failed to convert IDataList values: %v", err)
			return nil
		}
		pie.AddSeries(data.GetName(), convertedData)
	default:
		insyra.LogWarning("plot", "CreatePieChart", "Unsupported Data type: %T", config.Data)
		return nil
	}

	// 設置標籤和其他選項
	labelFormatter := "{b}: {c}" // 預設顯示名稱和值
	if config.ShowPercent && config.ShowLabels {
		labelFormatter = "{b}: {c} ({d}%)" // 增加百分比顯示
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
			Position:  config.LabelPos,
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

// convertToPieData 將 []float64 和 []string 轉換為 []opts.PieData
func convertToPieData(data []float64, labels []string) ([]opts.PieData, error) {
	if len(data) != len(labels) {
		return nil, errors.New("data length and label length do not match")
	}
	pieData := make([]opts.PieData, len(data))
	for i, value := range data {
		pieData[i] = opts.PieData{Name: labels[i], Value: value}
	}
	return pieData, nil
}
