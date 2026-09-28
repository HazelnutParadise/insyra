// gplot/step.go

package gplot

import (
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

// StepChartConfig defines the configuration for a multi-series step chart.
type StepChartConfig struct {
	Title     string    // Title of the chart.
	XAxis     []float64 // X-axis data.
	XAxisName string    // Optional: X-axis name.
	YAxisName string    // Optional: Y-axis name.
	StepStyle string    // Optional: Step style - "pre", "mid", "post". Default is "post".
}

// CreateStepChart draws one step line per list, named after the list. A plain
// slice is passed as insyra.NewDataList(values).SetName("name").
// config.StepStyle is "pre", "mid" or "post" (the default when empty); any
// other value is an error, so a misspelled style is not drawn as "post".
//
// When config.XAxis is nil, it is 0, 1, 2, ... up to the first list's length.
// It returns a nil chart and an error when no list is given, every one is
// nil, or any list cannot be drawn: its length differs from XAxis, it is
// empty, or it holds a NaN or an infinity. That error names each such list and
// why. A nil list among real ones is skipped with a warning.
//
// The values are read through DataList.ToF64Slice: a number, a fixed-point
// decimal included, is drawn as its value, and any other cell, whether nil,
// text (a numeric string such as "2" included) or a bool, is drawn as 0.
func CreateStepChart(config StepChartConfig, data ...insyra.IDataList) (*plot.Plot, error) {
	var stepKind plotter.StepKind
	switch config.StepStyle {
	case "pre":
		stepKind = plotter.PreStep
	case "mid":
		stepKind = plotter.MidStep
	case "post", "":
		stepKind = plotter.PostStep
	default:
		return nil, chartError("CreateStepChart", "unknown StepStyle %q; use \"pre\", \"mid\" or \"post\"", config.StepStyle)
	}
	return drawLines("CreateStepChart", config.Title, config.XAxis, config.XAxisName, config.YAxisName, stepKind, data)
}
