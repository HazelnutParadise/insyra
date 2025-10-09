package plot

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// LiquidChartConfig defines the configuration for a liquid chart.
type LiquidChartConfig struct {
	Title           string             // Title of the chart.
	Subtitle        string             // Subtitle of the chart.
	Data            map[string]float32 // Accepts map[string]float32.
	ShowLabels      bool               // Optional: Show labels on the liquid chart.
	IsWaveAnimation bool               // Optional: Enable/Disable wave animation.
	Shape           string             // Optional: Shape of the liquid chart (e.g., "diamond", "pin", "arrow", "triangle").
}

// CreateLiquidChart generates and returns a *charts.Liquid object based on LiquidChartConfig.
func CreateLiquidChart(config LiquidChartConfig) *charts.Liquid {
	liquid := charts.NewLiquid()

	// Set title and subtitle
	liquid.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Bottom: "0%",
		}),
	)

	// Sort Data by value in ascending order
	type seriesEntry struct {
		Name  string
		Value float32
	}

	var entries []seriesEntry
	for name, value := range config.Data {
		entries = append(entries, seriesEntry{Name: name, Value: value})
	}

	utils.ParallelSortStableFunc(entries, func(a, b seriesEntry) int {
		if a.Value > b.Value {
			return -1
		} else if a.Value < b.Value {
			return 1
		} else {
			return 0
		}
	}) // Track the highest value
	var maxValue float32

	// Process sorted Data
	for _, entry := range entries {
		if entry.Value > maxValue {
			maxValue = entry.Value
		}

		liquid.AddSeries(entry.Name, convertToLiquidData([]float32{entry.Value})).
			SetSeriesOptions(
				charts.WithLiquidChartOpts(opts.LiquidChart{
					IsWaveAnimation: opts.Bool(config.IsWaveAnimation),
					Shape:           config.Shape,
				}),
			)
	}

	// Show the highest value in the label
	if config.ShowLabels {
		liquid.SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Color:     "black",
				Formatter: fmt.Sprintf("%.1f%%", maxValue*100), // Display the maximum value
			}),
		)
	} else {
		liquid.SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show: opts.Bool(false),
			}),
		)
	}

	return liquid
}

// convertToLiquidData converts []float32 to []opts.LiquidData
func convertToLiquidData(data []float32) []opts.LiquidData {
	items := make([]opts.LiquidData, len(data))
	for i, value := range data {
		items[i] = opts.LiquidData{Value: value}
	}
	return items
}
