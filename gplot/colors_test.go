package gplot

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	gonumplot "gonum.org/v1/plot"
)

func hexOf(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

// Colors replaces the palette in every chart that has it, so an "Other" series
// can be gray with fewer than nine series. A short list repeats, as in plot.
func TestColorsReplaceThePalette(t *testing.T) {
	quietFatal(t)
	a := insyra.NewDataList(1.0, 2.0, 3.0).SetName("a")
	b := insyra.NewDataList(3.0, 1.0, 2.0).SetName("b")
	c := insyra.NewDataList(2.0, 3.0, 1.0).SetName("c")
	colors := []string{"#123456", "#7F8C8D"}
	charts := map[string]func() (*gonumplot.Plot, error){
		"bar":  func() (*gonumplot.Plot, error) { return CreateBarChart(BarChartConfig{Colors: colors}, a, b, c) },
		"line": func() (*gonumplot.Plot, error) { return CreateLineChart(LineChartConfig{Colors: colors}, a, b, c) },
		"step": func() (*gonumplot.Plot, error) { return CreateStepChart(StepChartConfig{Colors: colors}, a, b, c) },
		"scatter": func() (*gonumplot.Plot, error) {
			return CreateScatterChart(ScatterChartConfig{Colors: colors},
				ScatterSeries{Name: "a", X: a, Y: b}, ScatterSeries{Name: "b", X: b, Y: c}, ScatterSeries{Name: "c", X: c, Y: a})
		},
	}
	for name, build := range charts {
		t.Run(name, func(t *testing.T) {
			plt, err := build()
			if err != nil {
				t.Fatal(err)
			}
			svg := renderSVG(t, plt)
			for _, want := range colors {
				if !strings.Contains(svg, want) {
					t.Errorf("the chart does not use %s", want)
				}
			}
			// Series 3 wraps to the first colour instead of taking the palette's third.
			if third := hexOf(seriesColor(2)); strings.Contains(svg, third) {
				t.Errorf("the chart still uses the palette's %s", third)
			}
		})
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"#7F8C8D", "#7F8C8D", true},
		{"#7f8c8d", "#7F8C8D", true},
		{"#abc", "#AABBCC", true},
		{"#11223344", "", true},
		{"green", "#008000", true},
		{"Green", "#008000", true},
		{" gray ", "#808080", true},
		{"", "", false},
		{"#12", "", false},
		{"#12345", "", false},
		{"#xyzxyz", "", false},
		{"notacolour", "", false},
		{"rgb(1,2,3)", "", false},
	}
	for _, tt := range tests {
		c, ok := parseColor(tt.in)
		if ok != tt.ok {
			t.Errorf("parseColor(%q) ok = %v; want %v", tt.in, ok, tt.ok)
			continue
		}
		if ok && tt.want != "" && hexOf(c) != tt.want {
			t.Errorf("parseColor(%q) = %s; want %s", tt.in, hexOf(c), tt.want)
		}
	}
	c, _ := parseColor("#11223344")
	if got := color.NRGBAModel.Convert(c).(color.NRGBA); got != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x44}) {
		t.Errorf("#11223344 = %+v", got)
	}
}

func TestAnUnreadableColourIsAnError(t *testing.T) {
	quietFatal(t)
	a := insyra.NewDataList(1.0, 2.0).SetName("a")
	colors := []string{"#123456", "nope"}
	calls := map[string]func() (*gonumplot.Plot, error){
		"CreateBarChart":  func() (*gonumplot.Plot, error) { return CreateBarChart(BarChartConfig{Colors: colors}, a) },
		"CreateLineChart": func() (*gonumplot.Plot, error) { return CreateLineChart(LineChartConfig{Colors: colors}, a) },
		"CreateStepChart": func() (*gonumplot.Plot, error) { return CreateStepChart(StepChartConfig{Colors: colors}, a) },
		"CreateScatterChart": func() (*gonumplot.Plot, error) {
			return CreateScatterChart(ScatterChartConfig{Colors: colors}, ScatterSeries{Name: "a", X: a, Y: a})
		},
	}
	for fn, call := range calls {
		t.Run(fn, func(t *testing.T) {
			insyra.ClearErrors()
			plt, err := call()
			if err == nil || plt != nil {
				t.Fatalf("chart %v, error %v; want a nil chart and an error", plt, err)
			}
			for _, want := range []string{"gplot: " + fn, "Colors[1]", `"nope"`} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

// The constants are the palette, in order, so a config can name them.
func TestColorConstantsAreThePalette(t *testing.T) {
	want := []string{ColorBlue, ColorOrange, ColorRed, ColorTeal, ColorPurple, ColorPink, ColorGreen, ColorBrightBlue, ColorGray}
	for i, c := range want {
		if got := hexOf(seriesColor(i)); got != c {
			t.Errorf("series %d is %s; the constant says %s", i+1, got, c)
		}
	}
	if _, ok := parseColor(ColorCream); !ok {
		t.Errorf("ColorCream %q does not parse", ColorCream)
	}
}

func TestBackgroundColor(t *testing.T) {
	quietFatal(t)
	a := insyra.NewDataList(1.0, 2.0, 3.0).SetName("a")
	grid, err := insyra.ReadSlice2D([][]float64{{1, 2}, {3, 4}})
	if err != nil {
		t.Fatal(err)
	}
	const bg = ColorCream
	calls := map[string]func(string) (*gonumplot.Plot, error){
		"CreateBarChart": func(c string) (*gonumplot.Plot, error) { return CreateBarChart(BarChartConfig{BackgroundColor: c}, a) },
		"CreateLineChart": func(c string) (*gonumplot.Plot, error) {
			return CreateLineChart(LineChartConfig{BackgroundColor: c}, a)
		},
		"CreateStepChart": func(c string) (*gonumplot.Plot, error) {
			return CreateStepChart(StepChartConfig{BackgroundColor: c}, a)
		},
		"CreateScatterChart": func(c string) (*gonumplot.Plot, error) {
			return CreateScatterChart(ScatterChartConfig{BackgroundColor: c}, ScatterSeries{Name: "a", X: a, Y: a})
		},
		"CreateHistogram": func(c string) (*gonumplot.Plot, error) {
			return CreateHistogram(HistogramConfig{BackgroundColor: c}, a)
		},
		"CreateFunctionPlot": func(c string) (*gonumplot.Plot, error) {
			return CreateFunctionPlot(FunctionPlotConfig{BackgroundColor: c, XMax: 1}, func(x float64) float64 { return x })
		},
		"CreateHeatmapChart": func(c string) (*gonumplot.Plot, error) {
			return CreateHeatmapChart(HeatmapChartConfig{BackgroundColor: c}, grid)
		},
	}
	for fn, call := range calls {
		t.Run(fn, func(t *testing.T) {
			plt, err := call(bg)
			if err != nil {
				t.Fatal(err)
			}
			if got := hexOf(plt.BackgroundColor); got != bg {
				t.Errorf("background is %s; want %s", got, bg)
			}
			if svg := renderSVG(t, plt); !strings.Contains(svg, bg) {
				t.Errorf("the saved chart does not use %s", bg)
			}

			plt, err = call("")
			if err != nil {
				t.Fatal(err)
			}
			if got := hexOf(plt.BackgroundColor); got != "#FFFFFF" {
				t.Errorf("default background is %s; want white", got)
			}

			insyra.ClearErrors()
			plt, err = call("nope")
			if err == nil || plt != nil {
				t.Fatalf("chart %v, error %v; want a nil chart and an error", plt, err)
			}
			for _, want := range []string{"gplot: " + fn, "BackgroundColor", `"nope"`} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}
