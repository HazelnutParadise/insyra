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
// a zero-value config, a nil function, a ragged grid, a negative count.
// gplot takes tables now, which the table pads, but a ragged [][]float64
// reaches it through ReadSlice2D, so that path is still exercised.

// A zero-value BarChartConfig is the first thing anyone writes, and XAxis is
// the only field on it the documentation does not mark optional. It used to
// panic inside gonum's NominalX, which indexes names[0] with no length check.
// The bars are now numbered 1..n, the way plot.CreateBarChart numbers them.
func TestCreateBarChart_NoXAxisNumbersTheBars(t *testing.T) {
	quietFatal(t)

	// Values far from 1..3, so neither axis would print a tick "3" on its own:
	// the unlabelled numeric x axis runs 0..2 and the y axis counts in hundreds.
	plt, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(100, 200, 300))
	if err != nil {
		t.Fatalf("CreateBarChart refused valid data with no labels: %v", err)
	}

	// The generated labels reach the axis, so the rendered chart carries them.
	svg := filepath.Join(t.TempDir(), "bar.svg")
	if err := SaveChart(plt, svg); err != nil {
		t.Fatalf("the chart cannot be saved: %v", err)
	}
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
	quietFatal(t)

	for _, n := range []int{1, 5, 12} {
		values := make([]float64, n)
		for i := range values {
			values[i] = float64(i + 1)
		}
		plt, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(values))
		if err != nil {
			t.Fatalf("%d bars gave no chart: %v", n, err)
		}
		if err := SaveChart(plt, filepath.Join(t.TempDir(), "bar.png")); err != nil {
			t.Fatalf("%d bars: %v", n, err)
		}
	}
}

func TestCreateFunctionPlot_NilFunction(t *testing.T) {
	quietFatal(t)

	if plt, err := CreateFunctionPlot(FunctionPlotConfig{}, nil); plt != nil || err == nil {
		t.Error("CreateFunctionPlot did not refuse a nil function")
	}
	// A chart with explicit Y bounds takes a different path to the same call.
	if plt, err := CreateFunctionPlot(FunctionPlotConfig{YMin: -1, YMax: 1}, nil); plt != nil || err == nil {
		t.Error("CreateFunctionPlot did not refuse a nil function with Y bounds")
	}
}

func TestCreateHeatmapChart_RaggedGrid(t *testing.T) {
	quietFatal(t)

	tests := []struct {
		name string
		grid [][]float64
	}{
		{name: "short row after a long one", grid: [][]float64{{1, 2, 3}, {4}}},
		{name: "long row after a short one", grid: [][]float64{{1}, {2, 3}}},
		{name: "empty row in the middle", grid: [][]float64{{1, 2}, {}, {3, 4}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt, err := insyra.ReadSlice2D(tt.grid)
			if err != nil {
				t.Fatalf("ReadSlice2D: %v", err)
			}
			// ReadSlice2D pads a row shorter than the first with nil and drops
			// what a longer row holds past the first row's length (an AGENTS.md
			// follow-up), so the table is rectangular and the chart builds.
			plt, err := CreateHeatmapChart(HeatmapChartConfig{}, dt)
			if err != nil {
				t.Fatalf("CreateHeatmapChart: %v", err)
			}
			mustSave(t, plt, "heat.png")
		})
	}
}

func TestCreateHeatmapChart_NonPositiveColors(t *testing.T) {
	quietFatal(t)

	grid := insyra.NewDataTable(insyra.NewDataList(1, 3), insyra.NewDataList(2, 4))
	for _, colors := range []int{-1, -100} {
		plt, err := CreateHeatmapChart(HeatmapChartConfig{Colors: colors}, grid)
		if err != nil {
			t.Fatalf("Colors=%d gave no chart: %v", colors, err)
		}
		if err := SaveChart(plt, filepath.Join(t.TempDir(), "heat.png")); err != nil {
			t.Fatalf("Colors=%d: the chart cannot be saved: %v", colors, err)
		}
	}
}

// A DataTable that converts to a ragged grid takes the same path.
func TestCreateHeatmapChart_RaggedDataTable(t *testing.T) {
	quietFatal(t)

	dt := insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0),
		insyra.NewDataList(3.0),
	)
	// The table pads the shorter column with nil, which is drawn as 0.
	plt, err := CreateHeatmapChart(HeatmapChartConfig{}, dt)
	if err != nil {
		t.Fatalf("CreateHeatmapChart: %v", err)
	}
	mustSave(t, plt, "heat.png")
}
