// plot/heatmap.go

package plot

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// HeatmapChartConfig defines the configuration for a heatmap chart.
type HeatmapChartConfig struct {
	Title    string   // Title of the heatmap.
	Subtitle string   // Subtitle of the heatmap.
	XAxis    []string // X-axis data.
	YAxis    []string // Y-axis data (typically categories).
	Data     [][3]int // Heatmap data in the form of [x, y, value].
	Colors   []string // Optional: Colors for the heatmap. Default is ["#50a3ba", "#eac736", "#d94e5d"].
	Min      int      // Minimum value for visual map.
	Max      int      // Maximum value for visual map.
	GridTop  string   // Optional: Space between the top of the chart and the title. Default is "80".
}

// CreateHeatmapChart generates and returns a *charts.HeatMap object based on HeatmapChartConfig.
func CreateHeatmapChart(config HeatmapChartConfig) *charts.HeatMap {
	hm := charts.NewHeatMap()

	if config.GridTop == "" {
		config.GridTop = "80"
	}
	if config.Colors == nil {
		config.Colors = []string{"#50a3ba", "#eac736", "#d94e5d"}
	}

	// Set title and subtitle
	hm.SetGlobalOptions(
		charts.WithGridOpts(opts.Grid{
			Top: config.GridTop,
		}),
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type:      "category",
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type:      "category",
			Data:      config.YAxis,
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(true),
			Min:        float32(config.Min),
			Max:        float32(config.Max),
			InRange: &opts.VisualMapInRange{
				Color: config.Colors,
			},
			Orient: "horizontal",
			Bottom: "0%",
			Right:  "0%",
		}),
	)

	// Add heatmap data
	hm.SetXAxis(config.XAxis).AddSeries("heatmap", convertToHeatMapData(config.Data))
	return hm
}

// convertToHeatMapData converts the heatmap data into the format needed by go-echarts.
func convertToHeatMapData(data [][3]int) []opts.HeatMapData {
	items := make([]opts.HeatMapData, len(data))
	for i, v := range data {
		items[i] = opts.HeatMapData{Value: [3]interface{}{v[0], v[1], v[2]}}
	}
	return items
}
