package plot

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// chart-constructors: every constructor returns (chart, error). A chart it
// cannot build is a nil chart and an error that names the function; an
// outcome that still gives a chart is a warning with a nil error.

func TestEveryConstructorReturnsAChartAndAnError(t *testing.T) {
	errorType := reflect.TypeFor[error]()
	constructors := map[string]any{
		"CreateBarChart":        CreateBarChart,
		"CreateBoxPlot":         CreateBoxPlot,
		"CreateFunnelChart":     CreateFunnelChart,
		"CreateGaugeChart":      CreateGaugeChart,
		"CreateHeatMap":         CreateHeatMap[string, string],
		"CreateKlineChart":      CreateKlineChart,
		"CreateLineChart":       CreateLineChart,
		"CreatePieChart":        CreatePieChart,
		"CreateRadarChart":      CreateRadarChart,
		"CreateSankeyChart":     CreateSankeyChart,
		"CreateScatterChart":    CreateScatterChart,
		"CreateThemeRiverChart": CreateThemeRiverChart,
		"CreateWordCloud":       CreateWordCloud,
	}
	for name, fn := range constructors {
		ft := reflect.TypeOf(fn)
		if ft.NumOut() != 2 || ft.Out(1) != errorType {
			t.Errorf("%s returns %v; want (chart, error)", name, ft)
			continue
		}
		if !ft.Out(0).Implements(reflect.TypeFor[Renderable]()) {
			t.Errorf("%s: first result %v is not a Renderable", name, ft.Out(0))
		}
	}
}

// failure is one documented input a constructor refuses. call reports whether
// the chart came back nil, and the error.
type failure struct {
	name string
	fn   string
	call func() (bool, error)
}

func TestConstructorsReturnAnErrorForWhatTheyCannotDraw(t *testing.T) {
	quiet(t)

	var typedNil *insyra.DataList
	day := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)

	tests := []failure{
		{"bar with no lists", "CreateBarChart", func() (bool, error) {
			c, err := CreateBarChart(BarChartConfig{})
			return c == nil, err
		}},
		{"bar with only nil lists", "CreateBarChart", func() (bool, error) {
			c, err := CreateBarChart(BarChartConfig{}, nil, typedNil)
			return c == nil, err
		}},
		{"line with no lists", "CreateLineChart", func() (bool, error) {
			c, err := CreateLineChart(LineChartConfig{})
			return c == nil, err
		}},
		{"line with only nil lists", "CreateLineChart", func() (bool, error) {
			c, err := CreateLineChart(LineChartConfig{}, nil, typedNil)
			return c == nil, err
		}},
		{"scatter with an empty map", "CreateScatterChart", func() (bool, error) {
			c, err := CreateScatterChart(ScatterChartConfig{}, map[string][]ScatterPoint{})
			return c == nil, err
		}},
		{"pie with no items", "CreatePieChart", func() (bool, error) {
			c, err := CreatePieChart(PieChartConfig{})
			return c == nil, err
		}},
		{"box plot with no series", "CreateBoxPlot", func() (bool, error) {
			c, err := CreateBoxPlot(BoxPlotConfig{})
			return c == nil, err
		}},
		{"box plot whose lists are all nil", "CreateBoxPlot", func() (bool, error) {
			c, err := CreateBoxPlot(BoxPlotConfig{}, BoxPlotSeries{Name: "s", Data: []insyra.IDataList{nil, typedNil}})
			return c == nil, err
		}},
		{"heat map with no points", "CreateHeatMap", func() (bool, error) {
			c, err := CreateHeatMap[string, string](HeatMapConfig{})
			return c == nil, err
		}},
		{"calendar heat map with string x values", "CreateHeatMap", func() (bool, error) {
			c, err := CreateHeatMap(HeatMapConfig{UseCalendar: true}, NewHeatMapPoint("mon", "am", 1))
			return c == nil, err
		}},
		{"calendar heat map with no calendar options", "CreateHeatMap", func() (bool, error) {
			c, err := CreateHeatMap(HeatMapConfig{UseCalendar: true}, NewHeatMapPoint(day, "am", 1))
			return c == nil, err
		}},
		{"radar with no series", "CreateRadarChart", func() (bool, error) {
			c, err := CreateRadarChart(RadarChartConfig{}, nil)
			return c == nil, err
		}},
		{"radar with neither indicators nor maximums", "CreateRadarChart", func() (bool, error) {
			c, err := CreateRadarChart(RadarChartConfig{}, []RadarSeries{{Name: "s", Values: []float32{1}}})
			return c == nil, err
		}},
		{"kline with no points", "CreateKlineChart", func() (bool, error) {
			c, err := CreateKlineChart(KlineChartConfig{})
			return c == nil, err
		}},
		{"funnel with an empty map", "CreateFunnelChart", func() (bool, error) {
			c, err := CreateFunnelChart(FunnelChartConfig{}, map[string]float64{})
			return c == nil, err
		}},
		{"sankey with no links", "CreateSankeyChart", func() (bool, error) {
			c, err := CreateSankeyChart(SankeyChartConfig{})
			return c == nil, err
		}},
		{"theme river with no data", "CreateThemeRiverChart", func() (bool, error) {
			c, err := CreateThemeRiverChart(ThemeRiverChartConfig{})
			return c == nil, err
		}},
		{"word cloud with an empty list", "CreateWordCloud", func() (bool, error) {
			c, err := CreateWordCloud(WordCloudConfig{}, insyra.NewDataList())
			return c == nil, err
		}},
		{"word cloud with a nil list", "CreateWordCloud", func() (bool, error) {
			c, err := CreateWordCloud(WordCloudConfig{}, nil)
			return c == nil, err
		}},
		{"word cloud with a typed nil list", "CreateWordCloud", func() (bool, error) {
			c, err := CreateWordCloud(WordCloudConfig{}, typedNil)
			return c == nil, err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insyra.ClearErrors()
			isNil, err := tt.call()
			if err == nil {
				t.Fatal("no error")
			}
			if !isNil {
				t.Error("a chart came back together with the error")
			}
			prefix := "plot: " + tt.fn + ": "
			if !strings.HasPrefix(err.Error(), prefix) {
				t.Errorf("error %q does not start with %q", err, prefix)
			}
			// The error is the whole report: nothing is logged, not even a
			// warning about a nil list skipped on the way to failing.
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

func TestGaugeReturnsNoError(t *testing.T) {
	quiet(t)

	c, err := CreateGaugeChart(GaugeChartConfig{Title: "gauge"}, 42)
	if err != nil || c == nil {
		t.Fatalf("CreateGaugeChart(42) = %v, %v", c, err)
	}
}

// A nil list among real ones still gives a chart, with a warning naming it.
func TestANilListAmongRealOnesIsAWarning(t *testing.T) {
	quiet(t)
	good := numbers("s", 1, 2, 3)

	insyra.ClearErrors()
	c, err := CreateLineChart(LineChartConfig{Title: "t"}, nil, good)
	if err != nil || c == nil {
		t.Fatalf("CreateLineChart(nil, good) = %v, %v", c, err)
	}
	renderToHTML(t, c)
	if !loggedWarning("CreateLineChart", "data list 0 is nil") {
		t.Error("no warning names the dropped list")
	}
}

// A box plot series with no lists is dropped by name; when none is left the
// error says so instead of claiming no series was given.
func TestBoxPlotNamesTheSeriesItDrops(t *testing.T) {
	quiet(t)

	empty := BoxPlotSeries{Name: "empty"}
	good := BoxPlotSeries{Name: "good", Data: []insyra.IDataList{numbers("g", 1, 2, 3, 4, 5)}}

	insyra.ClearErrors()
	c, err := CreateBoxPlot(BoxPlotConfig{}, empty, good)
	if err != nil || c == nil {
		t.Fatalf("CreateBoxPlot(empty, good) = %v, %v", c, err)
	}
	if !loggedWarning("CreateBoxPlot", `"empty"`) {
		t.Error("no warning names the dropped series")
	}

	c, err = CreateBoxPlot(BoxPlotConfig{}, empty)
	if err == nil || c != nil {
		t.Fatalf("CreateBoxPlot(empty) = %v, %v; want a nil chart and an error", c, err)
	}
	if !strings.Contains(err.Error(), "no series has any data") {
		t.Errorf("error %q does not say that no series has data", err)
	}
}

// How a cell that is not a number is drawn, as Docs/plot.md describes it.
// These pin behaviour the ToF64Slice follow-up in AGENTS.md keeps as it is.
func TestHowCellsThatAreNotNumbersAreDrawn(t *testing.T) {
	quiet(t)

	t.Run("bar: one text cell makes the Y axis a category axis", func(t *testing.T) {
		c, err := CreateBarChart(BarChartConfig{}, numbers("s", 1, "abc", 3))
		if err != nil {
			t.Fatal(err)
		}
		html := renderToHTML(t, c)
		if !strings.Contains(html, `"yAxis":[{"type":"category","data":["1","abc","3"]`) {
			t.Errorf("the Y axis is not the categories 1, abc, 3:\n%s", html)
		}
		if !strings.Contains(html, `"data":[{"value":0},{"value":1},{"value":2}]`) {
			t.Error("the values are not drawn at their category positions")
		}
	})
	t.Run("line: a nil cell prints as <nil> and makes a category axis", func(t *testing.T) {
		c, err := CreateLineChart(LineChartConfig{}, numbers("s", 1, nil, 3))
		if err != nil {
			t.Fatal(err)
		}
		// go-echarts writes the options without HTML escaping, so <nil> is literal.
		if html := renderToHTML(t, c); !strings.Contains(html, `"data":["1","<nil>","3"]`) {
			t.Errorf("the Y axis is not the categories 1, <nil>, 3:\n%s", html)
		}
	})
	t.Run("bar: a numeric string keeps a value axis and is drawn as 0", func(t *testing.T) {
		c, err := CreateBarChart(BarChartConfig{}, numbers("s", 1, "2", 3))
		if err != nil {
			t.Fatal(err)
		}
		html := renderToHTML(t, c)
		if !strings.Contains(html, `"yAxis":[{"type":"value"}]`) {
			t.Error("the Y axis is not a value axis")
		}
		if !strings.Contains(html, `"data":[{"value":1},{"value":0},{"value":3}]`) {
			t.Error(`"2" is not drawn as 0`)
		}
	})
	t.Run("box plot: cells that are not numbers are left out of the summary", func(t *testing.T) {
		c, err := CreateBoxPlot(BoxPlotConfig{}, BoxPlotSeries{
			Name: "s",
			Data: []insyra.IDataList{numbers("g", 1, "abc", 3, nil, 9, "5")},
		})
		if err != nil {
			t.Fatal(err)
		}
		// Min, Q1, median, Q3 and max of 1, 3 and 9.
		if html := renderToHTML(t, c); !strings.Contains(html, `"value":[1,2,3,6,9]`) {
			t.Errorf("the summary is not that of 1, 3, 9:\n%s", html)
		}
	})
}

// loggedWarning reports whether fn logged a warning containing substr since
// the buffer was last cleared.
func loggedWarning(fn, substr string) bool {
	for _, rec := range insyra.GetAllErrors() {
		if rec.FuncName == fn && rec.Level == insyra.LogLevelWarning && strings.Contains(rec.Message, substr) {
			return true
		}
	}
	return false
}
