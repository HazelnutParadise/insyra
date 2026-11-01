package gplot

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	gonumplot "gonum.org/v1/plot"
)

// chart-constructors: plot and gplot take the same shapes under the same
// names (#413).

func TestScatterChartIsTheSharedName(t *testing.T) {
	quietFatal(t)
	s := ScatterSeries{Name: "s", X: insyra.NewDataList(1, 2, 3), Y: insyra.NewDataList(4, 5, 6)}
	plt, err := CreateScatterChart(ScatterChartConfig{Title: "t"}, s)
	if err != nil {
		t.Fatal(err)
	}
	mustSave(t, plt, "scatter.png")

	// The former names stay one release.
	old, err := CreateScatterPlot(ScatterPlotConfig{Title: "t"}, s)
	if err != nil || old == nil {
		t.Fatalf("CreateScatterPlot = %v, %v", old, err)
	}
}

func TestScatterChartRefusesACellThatIsNotANumber(t *testing.T) {
	quietFatal(t)
	xs := insyra.NewDataList(1, 2, 3)
	tests := []struct {
		name   string
		series ScatterSeries
		want   []string
	}{
		{"text in X", ScatterSeries{Name: "s", X: insyra.NewDataList(1, "two", 3), Y: xs}, []string{`"s"`, "X", "row index 1", "two"}},
		{"numeric string in Y", ScatterSeries{Name: "s", X: xs, Y: insyra.NewDataList(1, 2, "3")}, []string{`"s"`, "Y", "row index 2"}},
		{"nil cell in Y", ScatterSeries{Name: "s", X: xs, Y: insyra.NewDataList(nil, 2, 3)}, []string{`"s"`, "Y", "row index 0"}},
		{"bool in X", ScatterSeries{Name: "s", X: insyra.NewDataList(1, true, 3), Y: xs}, []string{`"s"`, "X", "row index 1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insyra.ClearErrors()
			plt, err := CreateScatterChart(ScatterChartConfig{}, tt.series)
			if err == nil || plt != nil {
				t.Fatalf("chart %v, error %v; want a nil chart and an error", plt, err)
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not mention %q", err, w)
				}
			}
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

func TestBarChartDrawsSeveralListsSideBySide(t *testing.T) {
	quietFatal(t)
	a := insyra.NewDataList(1.0, 2.0, 3.0).SetName("2025")
	b := insyra.NewDataList(2.0, 3.0, 4.0).SetName("2026")
	plt, err := CreateBarChart(BarChartConfig{XAxis: []string{"Q1", "Q2", "Q3"}}, a, b)
	if err != nil {
		t.Fatal(err)
	}
	svg := renderSVG(t, plt)
	for _, name := range []string{"2025", "2026"} {
		if !strings.Contains(svg, name) {
			t.Errorf("the legend does not name %q", name)
		}
	}

	// One list keeps the single-series chart: no legend entry.
	one, err := CreateBarChart(BarChartConfig{}, a)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(renderSVG(t, one), "2025") {
		t.Error("a single list got a legend")
	}
}

func TestBarChartRefusesWhatItCannotGroup(t *testing.T) {
	quietFatal(t)
	a := insyra.NewDataList(1.0, 2.0, 3.0).SetName("a")
	b := insyra.NewDataList(1.0, 2.0).SetName("b")
	tests := []struct {
		name string
		call func() (bool, error)
		want []string
	}{
		{"different lengths", func() (bool, error) {
			plt, err := CreateBarChart(BarChartConfig{}, a, b)
			return plt == nil, err
		}, []string{"list 2", "2 values", "3"}},
		{"error bars with two lists", func() (bool, error) {
			plt, err := CreateBarChart(BarChartConfig{ErrorBars: []float64{0.1, 0.1, 0.1}}, a, a)
			return plt == nil, err
		}, []string{"ErrorBars", "one list"}},
		{"an empty list among real ones", func() (bool, error) {
			plt, err := CreateBarChart(BarChartConfig{}, a, insyra.NewDataList())
			return plt == nil, err
		}, []string{"list 2", "empty"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insyra.ClearErrors()
			isNil, err := tt.call()
			if err == nil || !isNil {
				t.Fatalf("nil chart %v, error %v; want a nil chart and an error", isNil, err)
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not mention %q", err, w)
				}
			}
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

// wrapped embeds a list the way isr.DL does; a nil one inside must count as
// nil instead of panicking on its first method call.
type wrapped struct{ *insyra.DataList }

func TestANilListInsideAWrapperIsDropped(t *testing.T) {
	quietFatal(t)
	real := insyra.NewDataList(1.0, 2.0, 3.0).SetName("real")
	calls := map[string]func() error{
		"line": func() error { _, err := CreateLineChart(LineChartConfig{}, wrapped{}, real); return err },
		"step": func() error { _, err := CreateStepChart(StepChartConfig{}, wrapped{}, real); return err },
		"bar":  func() error { _, err := CreateBarChart(BarChartConfig{}, wrapped{}, real); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			insyra.ClearErrors()
			if err := call(); err != nil {
				t.Fatal(err)
			}
			if !loggedWarning("data list 1 is nil") {
				t.Errorf("no warning names data list 1: %+v", insyra.GetAllErrors())
			}
		})
	}

	insyra.ClearErrors()
	if _, err := CreateHistogram(HistogramConfig{}, wrapped{}); err == nil {
		t.Error("a histogram of a wrapped nil list did not fail")
	}
	if _, err := CreateScatterChart(ScatterChartConfig{}, ScatterSeries{Name: "s", X: wrapped{}, Y: real}); err == nil {
		t.Error("a scatter series with a wrapped nil X did not fail")
	}
}

// Every chart colours series i from the palette plot uses: series 1 is the
// same Insyra blue whether it is a bar, a line, a step or a set of points.
func TestEveryChartSharesOnePalette(t *testing.T) {
	quietFatal(t)
	a := insyra.NewDataList(1.0, 2.0, 3.0).SetName("a")
	b := insyra.NewDataList(3.0, 1.0, 2.0).SetName("b")
	rgb := func(c color.Color) string {
		r, g, bl, _ := c.RGBA()
		return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, bl>>8)
	}
	charts := map[string]func() (*gonumplot.Plot, error){
		"bar":  func() (*gonumplot.Plot, error) { return CreateBarChart(BarChartConfig{}, a, b) },
		"line": func() (*gonumplot.Plot, error) { return CreateLineChart(LineChartConfig{}, a, b) },
		"step": func() (*gonumplot.Plot, error) { return CreateStepChart(StepChartConfig{}, a, b) },
		"scatter": func() (*gonumplot.Plot, error) {
			return CreateScatterChart(ScatterChartConfig{}, ScatterSeries{Name: "a", X: a, Y: b}, ScatterSeries{Name: "b", X: b, Y: a})
		},
		"histogram": func() (*gonumplot.Plot, error) { return CreateHistogram(HistogramConfig{}, a) },
		"function": func() (*gonumplot.Plot, error) {
			return CreateFunctionPlot(FunctionPlotConfig{XMax: 1}, func(x float64) float64 { return x })
		},
	}
	want := map[string][]color.Color{
		"bar":       {seriesColor(0), seriesColor(1)},
		"line":      {seriesColor(0), seriesColor(1)},
		"step":      {seriesColor(0), seriesColor(1)},
		"scatter":   {seriesColor(0), seriesColor(1)},
		"histogram": {seriesColor(0)},
		"function":  {seriesColor(0)},
	}
	for name, build := range charts {
		t.Run(name, func(t *testing.T) {
			plt, err := build()
			if err != nil {
				t.Fatal(err)
			}
			svg := renderSVG(t, plt)
			for i, c := range want[name] {
				if !strings.Contains(svg, rgb(c)) {
					t.Errorf("series %d is not drawn in %s", i+1, rgb(c))
				}
			}
		})
	}
}

// A text cell from a file can carry terminal escape codes; the error quotes
// it so they reach the terminal as text.
func TestScatterChartQuotesTheCellItRefuses(t *testing.T) {
	quietFatal(t)
	s := ScatterSeries{Name: "s", X: insyra.NewDataList(1, "\x1b]52;c;cHduZWQ=\a", 3), Y: insyra.NewDataList(1, 2, 3)}
	_, err := CreateScatterChart(ScatterChartConfig{}, s)
	if err == nil {
		t.Fatal("no error")
	}
	if strings.ContainsAny(err.Error(), "\x1b\a") {
		t.Errorf("error %q carries raw control characters", err)
	}
	if !strings.Contains(err.Error(), `\x1b]52`) {
		t.Errorf("error %q does not show the escaped cell", err)
	}
}
