package internal

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type BaseChartConfig struct {
	Width           string // Width of the chart (default "900px").
	Height          string // Height of the chart (default "500px").
	BackgroundColor string // Background color of the chart (default "white").
	Theme           string // Theme of the chart.
	Title           string // Title of the chart.
	Subtitle        string // Subtitle of the chart.
	TitlePos        string // Title position: "top" or "bottom".
	HideLegend      bool   // Whether to hide the legend.
	LegendPos       string // Legend position: "top" or "bottom".
}

func SetBaseChartGlobalOptions[T any](chart Chart[T], config BaseChartConfig) {
	legendOpts := opts.Legend{
		Show: opts.Bool(!config.HideLegend),
	}
	switch config.LegendPos {
	case "top":
		legendOpts.Top = "top"
	case "bottom":
		legendOpts.Bottom = "bottom"
	case "left":
		legendOpts.Left = "left"
	case "right":
		legendOpts.Right = "right"
	}

	if config.BackgroundColor == "" {
		config.BackgroundColor = "white"
	}

	titleOpts := opts.Title{
		Title:             config.Title,
		Subtitle:          config.Subtitle,
		TextAlign:         "auto",
		TextVerticalAlign: "auto",
	}
	switch config.TitlePos {
	case "bottom":
		titleOpts.Bottom = "bottom"
	case "top":
		titleOpts.Top = "top"
	case "left":
		titleOpts.Left = "left"
	case "right":
		titleOpts.Right = "right"
	}

	// Set title and subtitle
	chart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:           config.Width,
			Height:          config.Height,
			BackgroundColor: config.BackgroundColor,
			Theme:           config.Theme,
			PageTitle:       config.Title,
		}),
		charts.WithTitleOpts(titleOpts),
		charts.WithLegendOpts(legendOpts),
		charts.WithGridOpts(opts.Grid{
			Top: "80px",
		}),
	)
}
