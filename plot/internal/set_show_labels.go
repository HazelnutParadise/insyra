package internal

import (
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func SetShowLabels[T any](chart Chart[T], showLabels bool, labelPos string, defaultLabelPos string) {
	if showLabels && labelPos == "" {
		labelPos = defaultLabelPos
	}
	chart.SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:     opts.Bool(showLabels),
			Position: labelPos,
		}),
	)
}
