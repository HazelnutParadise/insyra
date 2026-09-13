package gplot

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// error-philosophy says no insyra package panics under the default config.
// These four paths did. Every one is ordinary bad input, not something exotic:
// a zero-value config, a nil function, a short row, a negative count.

// A zero-value BarChartConfig is the first thing anyone writes, and XAxis is
// the only field on it the documentation does not mark optional. It used to
// panic inside gonum's NominalX, which indexes names[0] with no length check.
func TestCreateBarChart_NoXAxisStillDraws(t *testing.T) {
	quietLogs(t)

	plt := CreateBarChart(BarChartConfig{}, []float64{1, 2, 3})
	if plt == nil {
		t.Fatal("CreateBarChart returned nil for valid data with no labels")
	}
	mustSave(t, plt, "bar.png")
}

func TestCreateFunctionPlot_NilFunction(t *testing.T) {
	quietLogs(t)

	if plt := CreateFunctionPlot(FunctionPlotConfig{}, nil); plt != nil {
		t.Error("CreateFunctionPlot returned a chart for a nil function")
	}
	// A chart with explicit Y bounds takes a different path to the same call.
	if plt := CreateFunctionPlot(FunctionPlotConfig{YMin: -1, YMax: 1}, nil); plt != nil {
		t.Error("CreateFunctionPlot returned a chart for a nil function with Y bounds")
	}
}

// A row shorter than row 0 used to panic on an index out of range, because the
// grid's column count comes from row 0 alone.
func TestCreateHeatmapChart_ShortRow(t *testing.T) {
	quietLogs(t)

	tests := []struct {
		name string
		grid [][]float64
	}{
		{name: "short row after a long one", grid: [][]float64{{1, 2, 3}, {4}}},
		{name: "empty row in the middle", grid: [][]float64{{1, 2}, {}, {3, 4}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if plt := CreateHeatmapChart(HeatmapChartConfig{}, tt.grid); plt != nil {
				t.Error("CreateHeatmapChart returned a chart for a grid with a short row")
			}
		})
	}
}

// A row longer than row 0 never panicked: its extra values are not read. It
// keeps drawing.
func TestCreateHeatmapChart_LongRowStillDraws(t *testing.T) {
	quietLogs(t)

	mustSave(t, CreateHeatmapChart(HeatmapChartConfig{}, [][]float64{{1}, {2, 3}}), "heat.png")
}

func TestCreateHeatmapChart_NonPositiveColors(t *testing.T) {
	quietLogs(t)

	grid := [][]float64{{1, 2}, {3, 4}}
	for _, colors := range []int{-1, -100} {
		plt := CreateHeatmapChart(HeatmapChartConfig{Colors: colors}, grid)
		if plt == nil {
			t.Fatalf("Colors=%d gave no chart", colors)
		}
		mustSave(t, plt, "heat.png")
	}
}

// A DataTable that converts to a ragged grid takes the same path.
func TestCreateHeatmapChart_RaggedDataTable(t *testing.T) {
	quietLogs(t)

	dt := insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0),
		insyra.NewDataList(3.0),
	)
	// Whatever this converts to, it must not panic.
	_ = CreateHeatmapChart(HeatmapChartConfig{}, dt)
}
