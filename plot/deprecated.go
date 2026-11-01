package plot

import "github.com/go-echarts/go-echarts/v2/charts"

// HeatMapAxis is the former name of HeatmapAxis.
//
// Deprecated: use HeatmapAxis.
type HeatMapAxis = HeatmapAxis

// HeatMapConfig is the former name of HeatmapChartConfig.
//
// Deprecated: use HeatmapChartConfig.
type HeatMapConfig = HeatmapChartConfig

// HeatMapPoint is the former name of HeatmapPoint.
//
// Deprecated: use HeatmapPoint.
type HeatMapPoint[X HeatmapAxis, Y HeatmapAxis] = HeatmapPoint[X, Y]

// NewHeatMapPoint is the former name of NewHeatmapPoint.
//
// Deprecated: use NewHeatmapPoint.
func NewHeatMapPoint[X HeatmapAxis, Y HeatmapAxis](x X, y Y, value float64) HeatmapPoint[X, Y] {
	return NewHeatmapPoint(x, y, value)
}

// NewHeatMapMissingPoint is the former name of NewHeatmapMissingPoint.
//
// Deprecated: use NewHeatmapMissingPoint.
func NewHeatMapMissingPoint[X HeatmapAxis, Y HeatmapAxis](x X, y Y) HeatmapPoint[X, Y] {
	return NewHeatmapMissingPoint[X, Y](x, y)
}

// CreateHeatMap is the former name of CreateHeatmapChart.
//
// Deprecated: use CreateHeatmapChart.
func CreateHeatMap[X HeatmapAxis, Y HeatmapAxis](config HeatmapChartConfig, points ...HeatmapPoint[X, Y]) (*charts.HeatMap, error) {
	return CreateHeatmapChart(config, points...)
}
