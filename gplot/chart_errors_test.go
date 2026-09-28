package gplot

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	gonumplot "gonum.org/v1/plot"
)

// chart-constructors: every constructor returns (*plot.Plot, error) and takes
// insyra's own types, and a chart it cannot build is a nil chart and an error.

func TestConstructorSignatures(t *testing.T) {
	plotType := reflect.TypeFor[*gonumplot.Plot]()
	errorType := reflect.TypeFor[error]()
	listType := reflect.TypeFor[insyra.IDataList]()

	tests := []struct {
		name     string
		fn       any
		data     reflect.Type
		variadic bool
	}{
		{"CreateBarChart", CreateBarChart, listType, false},
		{"CreateHistogram", CreateHistogram, listType, false},
		{"CreateLineChart", CreateLineChart, reflect.SliceOf(listType), true},
		{"CreateStepChart", CreateStepChart, reflect.SliceOf(listType), true},
		{"CreateScatterPlot", CreateScatterPlot, reflect.TypeFor[[]ScatterSeries](), true},
		{"CreateHeatmapChart", CreateHeatmapChart, reflect.TypeFor[insyra.IDataTable](), false},
		{"CreateFunctionPlot", CreateFunctionPlot, reflect.TypeFor[func(float64) float64](), false},
	}
	for _, tt := range tests {
		ft := reflect.TypeOf(tt.fn)
		if ft.NumOut() != 2 || ft.Out(0) != plotType || ft.Out(1) != errorType {
			t.Errorf("%s returns %v; want (*plot.Plot, error)", tt.name, ft)
		}
		if ft.NumIn() != 2 || ft.In(1) != tt.data || ft.IsVariadic() != tt.variadic {
			t.Errorf("%s is %v; want its data as %v (variadic %v)", tt.name, ft, tt.data, tt.variadic)
		}
	}
}

func TestConstructorsReturnAnErrorForWhatTheyCannotDraw(t *testing.T) {
	quietFatal(t)

	var typedNil *insyra.DataList
	var typedNilTable *insyra.DataTable
	three := insyra.NewDataList(1.0, 2.0, 3.0).SetName("three")
	two := insyra.NewDataList(1.0, 2.0).SetName("two")
	withNaN := insyra.NewDataList(1.0, math.NaN(), 3.0).SetName("nan")

	tests := []struct {
		name string
		fn   string
		call func() (*gonumplot.Plot, error)
		want string // a fragment the error must carry
	}{
		{"bar with a nil list", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{}, nil)
		}, "no data"},
		{"bar with a typed nil list", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{}, typedNil)
		}, "no data"},
		{"bar with an empty list", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{}, insyra.NewDataList())
		}, ""},
		{"bar with NaN", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{}, withNaN)
		}, ""},
		{"bar with infinity", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, math.Inf(1)))
		}, ""},
		{"bar with error bars of the wrong length", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{ErrorBars: []float64{0.1}}, three)
		}, "ErrorBars has 1 values but the data has 3"},
		{"bar with a NaN error bar", "CreateBarChart", func() (*gonumplot.Plot, error) {
			return CreateBarChart(BarChartConfig{ErrorBars: []float64{0.1, math.NaN(), 0.3}}, three)
		}, "cannot draw the error bars"},
		{"histogram with a nil list", "CreateHistogram", func() (*gonumplot.Plot, error) {
			return CreateHistogram(HistogramConfig{}, nil)
		}, "no data"},
		{"histogram with an empty list", "CreateHistogram", func() (*gonumplot.Plot, error) {
			return CreateHistogram(HistogramConfig{}, insyra.NewDataList())
		}, ""},
		{"histogram with NaN", "CreateHistogram", func() (*gonumplot.Plot, error) {
			return CreateHistogram(HistogramConfig{}, withNaN)
		}, "index 1"},
		{"histogram with infinity", "CreateHistogram", func() (*gonumplot.Plot, error) {
			return CreateHistogram(HistogramConfig{}, insyra.NewDataList(1.0, 2.0, math.Inf(-1)))
		}, "index 2"},
		{"line with no lists", "CreateLineChart", func() (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{})
		}, "no data"},
		{"line with only nil lists", "CreateLineChart", func() (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{}, nil, typedNil)
		}, "no data"},
		{"line whose every series has the wrong length", "CreateLineChart", func() (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{XAxis: []float64{1, 2, 3}}, two)
		}, `cannot draw every series: series "two" has 2 values but XAxis has 3`},
		{"line whose only list is empty", "CreateLineChart", func() (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{}, insyra.NewDataList().SetName("empty"))
		}, `series "empty" has no values`},
		{"line whose only series holds NaN", "CreateLineChart", func() (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{}, withNaN)
		}, "cannot draw every series"},
		{"step with an unknown StepStyle", "CreateStepChart", func() (*gonumplot.Plot, error) {
			return CreateStepChart(StepChartConfig{StepStyle: "pr"}, three)
		}, `unknown StepStyle "pr"`},
		{"step with no lists", "CreateStepChart", func() (*gonumplot.Plot, error) {
			return CreateStepChart(StepChartConfig{})
		}, "no data"},
		{"step whose every series has the wrong length", "CreateStepChart", func() (*gonumplot.Plot, error) {
			return CreateStepChart(StepChartConfig{XAxis: []float64{1, 2, 3}}, two)
		}, "cannot draw every series"},
		{"scatter with no series", "CreateScatterPlot", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{})
		}, "no data"},
		{"scatter with a nil X", "CreateScatterPlot", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{}, ScatterSeries{Name: "s", Y: three})
		}, `"s"`},
		{"scatter with a typed nil Y", "CreateScatterPlot", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{}, ScatterSeries{Name: "s", X: three, Y: typedNil})
		}, `"s"`},
		{"scatter with X and Y of different lengths", "CreateScatterPlot", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{}, ScatterSeries{Name: "s", X: three, Y: two})
		}, "3"},
		{"scatter whose only series holds NaN", "CreateScatterPlot", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{}, ScatterSeries{Name: "s", X: three, Y: withNaN})
		}, `cannot draw every series: series "s" cannot be drawn`},
		{"scatter whose only series has no points", "CreateScatterPlot", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{}, ScatterSeries{Name: "s", X: insyra.NewDataList(), Y: insyra.NewDataList()})
		}, `series "s" has no points`},
		{"heat map with a nil table", "CreateHeatmapChart", func() (*gonumplot.Plot, error) {
			return CreateHeatmapChart(HeatmapChartConfig{}, nil)
		}, "no data"},
		{"heat map with a typed nil table", "CreateHeatmapChart", func() (*gonumplot.Plot, error) {
			return CreateHeatmapChart(HeatmapChartConfig{}, typedNilTable)
		}, "no data"},
		{"heat map with an empty table", "CreateHeatmapChart", func() (*gonumplot.Plot, error) {
			return CreateHeatmapChart(HeatmapChartConfig{}, insyra.NewDataTable())
		}, "empty"},
		{"heat map with an infinity", "CreateHeatmapChart", func() (*gonumplot.Plot, error) {
			return CreateHeatmapChart(HeatmapChartConfig{}, insyra.NewDataTable(
				insyra.NewDataList(1.0, math.Inf(1)), insyra.NewDataList(3.0, 4.0)))
		}, "row 1, column 0"},
		{"heat map of NaN alone", "CreateHeatmapChart", func() (*gonumplot.Plot, error) {
			return CreateHeatmapChart(HeatmapChartConfig{}, insyra.NewDataTable(insyra.NewDataList(math.NaN())))
		}, "every cell is NaN"},
		{"function plot with no function", "CreateFunctionPlot", func() (*gonumplot.Plot, error) {
			return CreateFunctionPlot(FunctionPlotConfig{}, nil)
		}, "no function"},
		{"function plot with an infinite X range", "CreateFunctionPlot", func() (*gonumplot.Plot, error) {
			return CreateFunctionPlot(FunctionPlotConfig{XMax: math.Inf(1)}, math.Sin)
		}, "XMax is +Inf"},
		{"function plot with a NaN Y bound", "CreateFunctionPlot", func() (*gonumplot.Plot, error) {
			return CreateFunctionPlot(FunctionPlotConfig{YMin: math.NaN(), YMax: 1}, math.Sin)
		}, "YMin is NaN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insyra.ClearErrors()
			plt, err := tt.call()
			if err == nil {
				t.Fatal("no error")
			}
			if plt != nil {
				t.Error("a chart came back together with the error")
			}
			if prefix := "gplot: " + tt.fn + ": "; !strings.HasPrefix(err.Error(), prefix) {
				t.Errorf("error %q does not start with %q", err, prefix)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
			// The error is the whole report: nothing is logged, not even the
			// warning ToF64Slice gives an empty list.
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

// A NaN among numbers is an empty cell, and the chart saves on every
// platform; only a grid of NaN alone is refused.
func TestHeatmapDrawsANaNAmongNumbers(t *testing.T) {
	quietFatal(t)

	plt, err := CreateHeatmapChart(HeatmapChartConfig{}, insyra.NewDataTable(
		insyra.NewDataList(1.0, math.NaN()), insyra.NewDataList(3.0, 4.0)))
	if err != nil {
		t.Fatalf("CreateHeatmapChart: %v", err)
	}
	mustSave(t, plt, "heat.png")
}

// gplot-refuses-partial-charts: a line, step or scatter chart that cannot draw
// every series it was given returns an error naming each such series, instead
// of a chart missing them. Nothing is logged for them.
func TestAnySeriesThatCannotBeDrawnFailsTheChart(t *testing.T) {
	quietFatal(t)

	three := insyra.NewDataList(1.0, 2.0, 3.0).SetName("three")

	tests := []struct {
		name string
		call func() (*gonumplot.Plot, error)
		want []string
	}{
		{"line: one series of two has the wrong length", func() (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{XAxis: []float64{1, 2, 3}}, three, insyra.NewDataList(1.0, 2.0).SetName("b"))
		}, []string{`series "b" has 2 values but XAxis has 3`}},
		{"step: an empty series and a NaN series beside a good one", func() (*gonumplot.Plot, error) {
			return CreateStepChart(StepChartConfig{}, three,
				insyra.NewDataList().SetName("empty"),
				insyra.NewDataList(1.0, math.NaN(), 3.0).SetName("nan"))
		}, []string{`series "empty" has 0 values but XAxis has 3`, `series "nan" cannot be drawn`}},
		{"scatter: an empty series beside a real one", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{},
				ScatterSeries{Name: "one", X: insyra.NewDataList(0, 1), Y: insyra.NewDataList(1, 2)},
				ScatterSeries{Name: "empty", X: insyra.NewDataList(), Y: insyra.NewDataList()})
		}, []string{`series "empty" has no points`}},
		{"scatter: a NaN series beside a real one", func() (*gonumplot.Plot, error) {
			return CreateScatterPlot(ScatterPlotConfig{},
				ScatterSeries{Name: "one", X: insyra.NewDataList(0, 1), Y: insyra.NewDataList(1, 2)},
				ScatterSeries{Name: "nan", X: insyra.NewDataList(0, 1), Y: insyra.NewDataList(1, math.NaN())})
		}, []string{`series "nan" cannot be drawn`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insyra.ClearErrors()
			plt, err := tt.call()
			if err == nil {
				t.Fatal("a chart missing a series came back with no error")
			}
			if plt != nil {
				t.Error("a chart came back together with the error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not say %q", err, want)
				}
			}
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

// A nil list is missing input, not a series that failed to draw: it is still
// dropped with a warning, the way plot drops one.
func TestANilListAmongRealOnesIsStillDropped(t *testing.T) {
	quietFatal(t)

	insyra.ClearErrors()
	plt, err := CreateStepChart(StepChartConfig{}, nil, insyra.NewDataList(1.0, 2.0, 3.0).SetName("three"))
	if err != nil {
		t.Fatalf("CreateStepChart: %v", err)
	}
	mustSave(t, plt, "step.png")
	if !loggedWarning("data list 0 is nil") {
		t.Error("no warning names the nil list")
	}
}

func TestScatterSeriesPairsXWithY(t *testing.T) {
	quietFatal(t)

	dt := insyra.NewDataTable(
		insyra.NewDataList(170, 175, 180).SetName("height"),
		insyra.NewDataList(70, 75, 80).SetName("weight"),
	)
	plt, err := CreateScatterPlot(ScatterPlotConfig{},
		ScatterSeries{Name: "people", X: dt.GetColByName("height"), Y: dt.GetColByName("weight")})
	if err != nil {
		t.Fatalf("CreateScatterPlot: %v", err)
	}
	if plt.X.Min != 170 || plt.X.Max != 180 || plt.Y.Min != 70 || plt.Y.Max != 80 {
		t.Errorf("axes span x [%v, %v], y [%v, %v]; want x [170, 180], y [70, 80]",
			plt.X.Min, plt.X.Max, plt.Y.Min, plt.Y.Max)
	}
	mustSave(t, plt, "scatter.png")
}

// A cell that is not a number is drawn as 0, the way ToF64Slice reads it, in
// every gplot chart. The comparison is between rendered files: the chart with
// such cells must be the chart of the same numbers with 0 in their place.
func TestCellsThatAreNotNumbersAreDrawnAsZero(t *testing.T) {
	quietFatal(t)

	t.Run("bar", func(t *testing.T) {
		mixed, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1, "2", nil, "abc", true))
		if err != nil {
			t.Fatal(err)
		}
		zeros, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, 0.0, 0.0, 0.0, 0.0))
		if err != nil {
			t.Fatal(err)
		}
		if renderSVG(t, mixed) != renderSVG(t, zeros) {
			t.Error(`the bars of 1, "2", nil, "abc", true differ from the bars of 1, 0, 0, 0, 0`)
		}
		// Control: the comparison sees a bar of 2 where the zeros have 0.
		two, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, 2.0, 0.0, 0.0, 0.0))
		if err != nil {
			t.Fatal(err)
		}
		if renderSVG(t, two) == renderSVG(t, zeros) {
			t.Fatal("the rendered files do not show the values, so this comparison proves nothing")
		}
	})

	t.Run("heat map reads every numeric type", func(t *testing.T) {
		typed := insyra.NewDataTable(
			insyra.NewDataList(int8(1), int8(2)),
			insyra.NewDataList(uint16(3), uint16(4)),
			insyra.NewDataList(float32(5), float32(6)),
		)
		floats := insyra.NewDataTable(
			insyra.NewDataList(1.0, 2.0),
			insyra.NewDataList(3.0, 4.0),
			insyra.NewDataList(5.0, 6.0),
		)
		a, err := CreateHeatmapChart(HeatmapChartConfig{}, typed)
		if err != nil {
			t.Fatal(err)
		}
		b, err := CreateHeatmapChart(HeatmapChartConfig{}, floats)
		if err != nil {
			t.Fatal(err)
		}
		if renderSVG(t, a) != renderSVG(t, b) {
			t.Error("the int8/uint16/float32 table is not drawn like the same values as float64")
		}
		// Control: the int8 and uint16 columns drawn as 0, which is what the
		// old conversion did, render differently.
		old, err := CreateHeatmapChart(HeatmapChartConfig{}, insyra.NewDataTable(
			insyra.NewDataList(0.0, 0.0),
			insyra.NewDataList(0.0, 0.0),
			insyra.NewDataList(5.0, 6.0),
		))
		if err != nil {
			t.Fatal(err)
		}
		if renderSVG(t, old) == renderSVG(t, b) {
			t.Fatal("the rendered files do not show the values, so this comparison proves nothing")
		}
	})

	t.Run("heat map draws text and nil as 0", func(t *testing.T) {
		mixed := insyra.NewDataTable(
			insyra.NewDataList(1.0, "x"),
			insyra.NewDataList(nil, 4.0),
		)
		zeros := insyra.NewDataTable(
			insyra.NewDataList(1.0, 0.0),
			insyra.NewDataList(0.0, 4.0),
		)
		a, err := CreateHeatmapChart(HeatmapChartConfig{}, mixed)
		if err != nil {
			t.Fatal(err)
		}
		b, err := CreateHeatmapChart(HeatmapChartConfig{}, zeros)
		if err != nil {
			t.Fatal(err)
		}
		if renderSVG(t, a) != renderSVG(t, b) {
			t.Error("text and nil cells are not drawn as 0")
		}
	})
}

func renderSVG(t *testing.T, plt *gonumplot.Plot) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "chart.svg")
	if err := SaveChart(plt, path); err != nil {
		t.Fatalf("SaveChart: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// loggedWarning reports whether a gplot warning containing substr was logged
// since the buffer was last cleared.
func loggedWarning(substr string) bool {
	for _, rec := range insyra.GetAllErrors() {
		if rec.PackageName == "gplot" && rec.Level == insyra.LogLevelWarning && strings.Contains(rec.Message, substr) {
			return true
		}
	}
	return false
}
