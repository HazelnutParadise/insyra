package gplot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// error-philosophy says no insyra package panics under the default config.
// These four paths did. Every one is ordinary bad input, not something exotic:
// a zero-value config, a nil function, a short row, a negative count.

// A zero-value BarChartConfig is the first thing anyone writes, and XAxis is
// the only field on it the documentation does not mark optional. It used to
// panic inside gonum's NominalX, which indexes names[0] with no length check.
// The bars are now numbered 1..n, the way plot.CreateBarChart numbers them.
func TestCreateBarChart_NoXAxisNumbersTheBars(t *testing.T) {
	quietLogs(t)

	// Values far from 1..3, so neither axis would print a tick "3" on its own:
	// the unlabelled numeric x axis runs 0..2 and the y axis counts in hundreds.
	plt := CreateBarChart(BarChartConfig{}, []float64{100, 200, 300})
	if plt == nil {
		t.Fatal("CreateBarChart returned nil for valid data with no labels")
	}

	// The generated labels reach the axis, so the rendered chart carries them.
	// SaveChart returns nothing on this line, so the file is read back instead.
	svg := filepath.Join(t.TempDir(), "bar.svg")
	SaveChart(plt, svg)
	b, err := os.ReadFile(svg)
	if err != nil {
		t.Fatalf("reading the chart back: %v", err)
	}
	for _, want := range []string{">1<", ">2<", ">3<"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the rendered chart has no tick %q", want)
		}
	}
}

// One label per data point, however many there are.
func TestCreateBarChart_NoXAxisMatchesTheDataLength(t *testing.T) {
	quietLogs(t)

	for _, n := range []int{1, 5, 12} {
		values := make([]float64, n)
		for i := range values {
			values[i] = float64(i + 1)
		}
		plt := CreateBarChart(BarChartConfig{}, values)
		if plt == nil {
			t.Fatalf("%d bars gave no chart", n)
		}
		mustSave(t, plt, "bar.png")
	}
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
