package gplot

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	gonumplot "gonum.org/v1/plot"
)

// Before this file only CreateBarChart and CreateHistogram had ever been called
// by a test, and step_test.go built step charts without ever rendering one. A
// chart that builds but cannot be written is not a working chart, so every case
// here goes all the way to a file on disk.

// mustSave writes the chart and asserts a non-empty file appears.
func mustSave(t *testing.T, plt *gonumplot.Plot, filename string) {
	t.Helper()
	if plt == nil {
		t.Fatal("the chart is nil")
	}
	path := filepath.Join(t.TempDir(), filename)
	if err := SaveChart(plt, path); err != nil {
		t.Fatalf("SaveChart(%s): %v", filename, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", filename, err)
	}
	if info.Size() == 0 {
		t.Errorf("%s was written but is empty", filename)
	}
}

func TestCreateBarChart_RendersEveryInputForm(t *testing.T) {
	quietFatal(t)

	config := BarChartConfig{
		Title:     "bars",
		XAxis:     []string{"a", "b", "c"},
		XAxisName: "x",
		YAxisName: "y",
		BarWidth:  15,
	}

	t.Run("float slice", func(t *testing.T) {
		mustSave(t, CreateBarChart(config, []float64{1, 2, 3}), "bar.png")
	})
	t.Run("DataList", func(t *testing.T) {
		mustSave(t, CreateBarChart(config, insyra.NewDataList(1.0, 2.0, 3.0)), "bar.png")
	})
	t.Run("IDataList", func(t *testing.T) {
		var dl insyra.IDataList = insyra.NewDataList(1.0, 2.0, 3.0)
		mustSave(t, CreateBarChart(config, dl), "bar.png")
	})
	t.Run("with error bars", func(t *testing.T) {
		c := config
		c.ErrorBars = []float64{0.1, 0.2, 0.3}
		mustSave(t, CreateBarChart(c, []float64{1, 2, 3}), "bar.png")
	})
	// Error bars of the wrong length are dropped with a warning; the chart is
	// still built rather than refused.
	t.Run("error bars of the wrong length", func(t *testing.T) {
		c := config
		c.ErrorBars = []float64{0.1}
		mustSave(t, CreateBarChart(c, []float64{1, 2, 3}), "bar.png")
	})
}

func TestCreateBarChart_RefusesWhatItCannotPlot(t *testing.T) {
	quietFatal(t)

	config := BarChartConfig{XAxis: []string{"a", "b"}}
	tests := []struct {
		name string
		data any
	}{
		{name: "unsupported type", data: "not data"},
		{name: "empty slice", data: []float64{}},
		{name: "NaN", data: []float64{1, math.NaN()}},
		{name: "infinity", data: []float64{1, math.Inf(1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if plt := CreateBarChart(config, tt.data); plt != nil {
				t.Errorf("CreateBarChart returned a chart for %v", tt.data)
			}
		})
	}
}

func TestCreateLineChart_RendersEveryInputForm(t *testing.T) {
	quietFatal(t)

	config := LineChartConfig{Title: "lines", XAxisName: "x", YAxisName: "y"}

	t.Run("map of series", func(t *testing.T) {
		mustSave(t, CreateLineChart(config, map[string][]float64{
			"one": {1, 2, 3},
			"two": {3, 2, 1},
		}), "line.png")
	})
	t.Run("DataLists", func(t *testing.T) {
		mustSave(t, CreateLineChart(config, []*insyra.DataList{
			insyra.NewDataList(1.0, 2.0, 3.0).SetName("one"),
			insyra.NewDataList(3.0, 2.0, 1.0).SetName("two"),
		}), "line.png")
	})
	t.Run("IDataLists", func(t *testing.T) {
		mustSave(t, CreateLineChart(config, []insyra.IDataList{
			insyra.NewDataList(1.0, 2.0, 3.0).SetName("one"),
		}), "line.png")
	})
	t.Run("explicit x axis", func(t *testing.T) {
		c := config
		c.XAxis = []float64{10, 20, 30}
		mustSave(t, CreateLineChart(c, map[string][]float64{"one": {1, 2, 3}}), "line.png")
	})

	if plt := CreateLineChart(config, "not data"); plt != nil {
		t.Error("CreateLineChart returned a chart for an unsupported type")
	}
}

// A series whose length does not match the x axis is skipped, and the chart is
// still returned. Rendering it proves the skip leaves the plot in one piece.
func TestCreateLineChart_SkipsAMismatchedSeries(t *testing.T) {
	quietFatal(t)

	config := LineChartConfig{XAxis: []float64{1, 2, 3}}
	plt := CreateLineChart(config, map[string][]float64{
		"good": {1, 2, 3},
		"bad":  {1, 2},
	})
	mustSave(t, plt, "line.png")
}

func TestCreateStepChart_RendersEveryStyle(t *testing.T) {
	quietFatal(t)

	for _, style := range []string{"pre", "mid", "post", "", "nonsense"} {
		t.Run("style "+style, func(t *testing.T) {
			plt := CreateStepChart(StepChartConfig{
				Title:     "steps",
				StepStyle: style,
			}, map[string][]float64{"one": {1, 3, 2}})
			mustSave(t, plt, "step.png")
		})
	}
}

func TestCreateScatterPlot_RendersEveryInputForm(t *testing.T) {
	quietFatal(t)

	config := ScatterPlotConfig{Title: "scatter", XAxisName: "x", YAxisName: "y"}

	t.Run("map of point pairs", func(t *testing.T) {
		mustSave(t, CreateScatterPlot(config, map[string][][]float64{
			"one": {{0, 1}, {1, 2}, {2, 4}},
		}), "scatter.png")
	})
	// A flat DataList is read as alternating x and y values.
	t.Run("DataLists", func(t *testing.T) {
		mustSave(t, CreateScatterPlot(config, []*insyra.DataList{
			insyra.NewDataList(0.0, 1.0, 1.0, 2.0, 2.0, 4.0).SetName("one"),
		}), "scatter.png")
	})
	// An odd trailing value has no partner and is dropped.
	t.Run("odd number of values", func(t *testing.T) {
		mustSave(t, CreateScatterPlot(config, []*insyra.DataList{
			insyra.NewDataList(0.0, 1.0, 1.0).SetName("one"),
		}), "scatter.png")
	})
	t.Run("IDataLists", func(t *testing.T) {
		mustSave(t, CreateScatterPlot(config, []insyra.IDataList{
			insyra.NewDataList(0.0, 1.0, 1.0, 2.0).SetName("one"),
		}), "scatter.png")
	})

	if plt := CreateScatterPlot(config, 42); plt != nil {
		t.Error("CreateScatterPlot returned a chart for an unsupported type")
	}
}

func TestCreateHistogram_RendersAndDefaultsBins(t *testing.T) {
	quietFatal(t)

	data := []float64{1, 2, 2, 3, 3, 3, 4, 5}

	t.Run("default bins", func(t *testing.T) {
		mustSave(t, CreateHistogram(HistogramConfig{Title: "hist"}, data), "hist.png")
	})
	t.Run("explicit bins", func(t *testing.T) {
		mustSave(t, CreateHistogram(HistogramConfig{Bins: 3}, data), "hist.png")
	})
	t.Run("DataList", func(t *testing.T) {
		mustSave(t, CreateHistogram(HistogramConfig{}, insyra.NewDataList(1.0, 2.0, 3.0)), "hist.png")
	})

	if plt := CreateHistogram(HistogramConfig{}, "not data"); plt != nil {
		t.Error("CreateHistogram returned a chart for an unsupported type")
	}
}

func TestCreateFunctionPlot_Renders(t *testing.T) {
	quietFatal(t)

	t.Run("default range", func(t *testing.T) {
		mustSave(t, CreateFunctionPlot(FunctionPlotConfig{Title: "f"}, func(x float64) float64 {
			return x * x
		}), "func.png")
	})
	t.Run("explicit ranges", func(t *testing.T) {
		mustSave(t, CreateFunctionPlot(FunctionPlotConfig{
			XMin: -2, XMax: 2, YMin: -1, YMax: 5,
			XAxisName: "x", YAxisName: "y",
		}, math.Sin), "func.png")
	})
}

func TestCreateHeatmapChart_Renders(t *testing.T) {
	quietFatal(t)

	grid := [][]float64{{1, 2, 3}, {4, 5, 6}}

	t.Run("float grid", func(t *testing.T) {
		mustSave(t, CreateHeatmapChart(HeatmapChartConfig{Title: "heat"}, grid), "heat.png")
	})
	t.Run("with axes and colours", func(t *testing.T) {
		mustSave(t, CreateHeatmapChart(HeatmapChartConfig{
			XAxis:  []float64{0, 1, 2},
			YAxis:  []float64{0, 1},
			Colors: 5,
			Alpha:  0.5,
		}, grid), "heat.png")
	})
	t.Run("DataTable", func(t *testing.T) {
		dt := insyra.NewDataTable(
			insyra.NewDataList(1.0, 4.0),
			insyra.NewDataList(2.0, 5.0),
		)
		mustSave(t, CreateHeatmapChart(HeatmapChartConfig{}, dt), "heat.png")
	})

	for _, tt := range []struct {
		name string
		data any
	}{
		{name: "unsupported type", data: "not data"},
		{name: "empty grid", data: [][]float64{}},
		{name: "empty first row", data: [][]float64{{}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if plt := CreateHeatmapChart(HeatmapChartConfig{}, tt.data); plt != nil {
				t.Errorf("CreateHeatmapChart returned a chart for %v", tt.data)
			}
		})
	}
}

// The extension chooses the writer. Only SaveChart's own error path had a test.
func TestSaveChart_Formats(t *testing.T) {
	quietFatal(t)

	plt := CreateBarChart(BarChartConfig{XAxis: []string{"a", "b"}}, []float64{1, 2})
	if plt == nil {
		t.Fatal("CreateBarChart returned nil")
	}

	dir := t.TempDir()
	for _, ext := range []string{"png", "svg", "pdf", "jpg", "tif"} {
		t.Run(ext, func(t *testing.T) {
			path := filepath.Join(dir, "chart."+ext)
			if err := SaveChart(plt, path); err != nil {
				t.Fatalf("SaveChart(.%s): %v", ext, err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if info.Size() == 0 {
				t.Errorf(".%s was written but is empty", ext)
			}
		})
	}
}

func TestSaveChart_NilChart(t *testing.T) {
	quietFatal(t)

	err := SaveChart(nil, filepath.Join(t.TempDir(), "chart.png"))
	if err == nil {
		t.Fatal("SaveChart(nil, …) returned no error")
	}
	if !strings.Contains(err.Error(), "no plot to save") {
		t.Errorf("error %q does not say there was no plot to save", err)
	}
}

func TestSaveChart_UnsupportedFormat(t *testing.T) {
	quietFatal(t)

	plt := CreateBarChart(BarChartConfig{XAxis: []string{"a"}}, []float64{1})
	err := SaveChart(plt, filepath.Join(t.TempDir(), "chart.bmp"))
	if err == nil {
		t.Fatal("SaveChart returned no error for an unsupported extension")
	}
}
