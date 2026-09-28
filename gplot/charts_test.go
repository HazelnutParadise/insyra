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

// built fails the test when a constructor returns an error, and hands back
// the chart: mustSave(t, built(t)(CreateBarChart(...)), "bar.png").
func built(t *testing.T) func(*gonumplot.Plot, error) *gonumplot.Plot {
	return func(plt *gonumplot.Plot, err error) *gonumplot.Plot {
		t.Helper()
		if err != nil {
			t.Fatalf("building the chart: %v", err)
		}
		return plt
	}
}

func TestCreateBarChart_Renders(t *testing.T) {
	quietFatal(t)

	config := BarChartConfig{
		Title:     "bars",
		XAxis:     []string{"a", "b", "c"},
		XAxisName: "x",
		YAxisName: "y",
		BarWidth:  15,
	}
	data := insyra.NewDataList(1.0, 2.0, 3.0)

	t.Run("DataList", func(t *testing.T) {
		mustSave(t, built(t)(CreateBarChart(config, data)), "bar.png")
	})
	t.Run("with error bars", func(t *testing.T) {
		c := config
		c.ErrorBars = []float64{0.1, 0.2, 0.3}
		mustSave(t, built(t)(CreateBarChart(c, data)), "bar.png")
	})
	// Error bars of the wrong length are dropped with a warning; the chart is
	// still built rather than refused.
	t.Run("error bars of the wrong length", func(t *testing.T) {
		c := config
		c.ErrorBars = []float64{0.1}
		mustSave(t, built(t)(CreateBarChart(c, data)), "bar.png")
	})
}

func TestCreateLineChart_Renders(t *testing.T) {
	quietFatal(t)

	config := LineChartConfig{Title: "lines", XAxisName: "x", YAxisName: "y"}

	t.Run("several lists", func(t *testing.T) {
		mustSave(t, built(t)(CreateLineChart(config,
			insyra.NewDataList(1.0, 2.0, 3.0).SetName("one"),
			insyra.NewDataList(3.0, 2.0, 1.0).SetName("two"),
		)), "line.png")
	})
	t.Run("a slice of lists", func(t *testing.T) {
		lists := []insyra.IDataList{insyra.NewDataList(1.0, 2.0, 3.0).SetName("one")}
		mustSave(t, built(t)(CreateLineChart(config, lists...)), "line.png")
	})
	t.Run("explicit x axis", func(t *testing.T) {
		c := config
		c.XAxis = []float64{10, 20, 30}
		mustSave(t, built(t)(CreateLineChart(c, insyra.NewDataList(1, 2, 3).SetName("one"))), "line.png")
	})
}

// A series whose length does not match the x axis is skipped, and the chart is
// still returned. Rendering it proves the skip leaves the plot in one piece.
func TestCreateLineChart_SkipsAMismatchedSeries(t *testing.T) {
	quietFatal(t)

	config := LineChartConfig{XAxis: []float64{1, 2, 3}}
	plt := built(t)(CreateLineChart(config,
		insyra.NewDataList(1, 2, 3).SetName("good"),
		insyra.NewDataList(1, 2).SetName("bad"),
	))
	mustSave(t, plt, "line.png")
}

func TestCreateStepChart_RendersEveryStyle(t *testing.T) {
	quietFatal(t)

	for _, style := range []string{"pre", "mid", "post", "", "nonsense"} {
		t.Run("style "+style, func(t *testing.T) {
			plt := built(t)(CreateStepChart(StepChartConfig{
				Title:     "steps",
				StepStyle: style,
			}, insyra.NewDataList(1, 3, 2).SetName("one")))
			mustSave(t, plt, "step.png")
		})
	}
}

func TestCreateScatterPlot_Renders(t *testing.T) {
	quietFatal(t)

	config := ScatterPlotConfig{Title: "scatter", XAxisName: "x", YAxisName: "y"}

	t.Run("one series", func(t *testing.T) {
		mustSave(t, built(t)(CreateScatterPlot(config, ScatterSeries{
			Name: "one",
			X:    insyra.NewDataList(0, 1, 2),
			Y:    insyra.NewDataList(1, 2, 4),
		})), "scatter.png")
	})
	t.Run("several series", func(t *testing.T) {
		mustSave(t, built(t)(CreateScatterPlot(config,
			ScatterSeries{Name: "one", X: insyra.NewDataList(0, 1), Y: insyra.NewDataList(1, 2)},
			ScatterSeries{Name: "two", X: insyra.NewDataList(2, 3), Y: insyra.NewDataList(4, 1)},
		)), "scatter.png")
	})
	// A series with no points is skipped; the others are drawn.
	t.Run("an empty series beside a real one", func(t *testing.T) {
		mustSave(t, built(t)(CreateScatterPlot(config,
			ScatterSeries{Name: "empty", X: insyra.NewDataList(), Y: insyra.NewDataList()},
			ScatterSeries{Name: "one", X: insyra.NewDataList(0, 1), Y: insyra.NewDataList(1, 2)},
		)), "scatter.png")
	})
}

func TestCreateHistogram_RendersAndDefaultsBins(t *testing.T) {
	quietFatal(t)

	data := insyra.NewDataList(1, 2, 2, 3, 3, 3, 4, 5)

	t.Run("default bins", func(t *testing.T) {
		mustSave(t, built(t)(CreateHistogram(HistogramConfig{Title: "hist"}, data)), "hist.png")
	})
	t.Run("explicit bins", func(t *testing.T) {
		mustSave(t, built(t)(CreateHistogram(HistogramConfig{Bins: 3}, data)), "hist.png")
	})
}

func TestCreateFunctionPlot_Renders(t *testing.T) {
	quietFatal(t)

	t.Run("default range", func(t *testing.T) {
		mustSave(t, built(t)(CreateFunctionPlot(FunctionPlotConfig{Title: "f"}, func(x float64) float64 {
			return x * x
		})), "func.png")
	})
	t.Run("explicit ranges", func(t *testing.T) {
		mustSave(t, built(t)(CreateFunctionPlot(FunctionPlotConfig{
			XMin: -2, XMax: 2, YMin: -1, YMax: 5,
			XAxisName: "x", YAxisName: "y",
		}, math.Sin)), "func.png")
	})
}

func TestCreateHeatmapChart_Renders(t *testing.T) {
	quietFatal(t)

	table := insyra.NewDataTable(
		insyra.NewDataList(1.0, 4.0),
		insyra.NewDataList(2.0, 5.0),
		insyra.NewDataList(3.0, 6.0),
	)

	t.Run("table", func(t *testing.T) {
		mustSave(t, built(t)(CreateHeatmapChart(HeatmapChartConfig{Title: "heat"}, table)), "heat.png")
	})
	t.Run("with axes and colours", func(t *testing.T) {
		mustSave(t, built(t)(CreateHeatmapChart(HeatmapChartConfig{
			XAxis:  []float64{0, 1, 2},
			YAxis:  []float64{0, 1},
			Colors: 5,
			Alpha:  0.5,
		}, table)), "heat.png")
	})
	// The documented way to draw a [][]float64 grid.
	t.Run("grid through ReadSlice2D", func(t *testing.T) {
		dt, err := insyra.ReadSlice2D([][]float64{{1, 2, 3}, {4, 5, 6}})
		if err != nil {
			t.Fatal(err)
		}
		mustSave(t, built(t)(CreateHeatmapChart(HeatmapChartConfig{}, dt)), "heat.png")
	})
}

// The extension chooses the writer. Only SaveChart's own error path had a test.
func TestSaveChart_Formats(t *testing.T) {
	quietFatal(t)

	plt := built(t)(CreateBarChart(BarChartConfig{XAxis: []string{"a", "b"}}, insyra.NewDataList(1, 2)))

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

	plt := built(t)(CreateBarChart(BarChartConfig{XAxis: []string{"a"}}, insyra.NewDataList(1)))
	err := SaveChart(plt, filepath.Join(t.TempDir(), "chart.bmp"))
	if err == nil {
		t.Fatal("SaveChart returned no error for an unsupported extension")
	}
}
