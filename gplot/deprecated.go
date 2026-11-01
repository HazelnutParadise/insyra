package gplot

import "gonum.org/v1/plot"

// ScatterPlotConfig is the former name of ScatterChartConfig.
//
// Deprecated: use ScatterChartConfig.
type ScatterPlotConfig = ScatterChartConfig

// CreateScatterPlot is the former name of CreateScatterChart.
//
// Deprecated: use CreateScatterChart.
func CreateScatterPlot(config ScatterChartConfig, series ...ScatterSeries) (*plot.Plot, error) {
	return CreateScatterChart(config, series...)
}
