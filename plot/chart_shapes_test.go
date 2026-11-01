package plot

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// chart-constructors: plot and gplot take the same shapes under the same
// names (#413).

func TestScatterChartTakesSeriesInArgumentOrder(t *testing.T) {
	quiet(t)
	c, err := CreateScatterChart(ScatterChartConfig{},
		ScatterSeries{Name: "b", X: insyra.NewDataList(1, 2, 3), Y: insyra.NewDataList(4, 5, 6)},
		ScatterSeries{Name: "a", X: insyra.NewDataList(1.5), Y: insyra.NewDataList(2.5)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.MultiSeries) != 2 || c.MultiSeries[0].Name != "b" || c.MultiSeries[1].Name != "a" {
		t.Fatalf("series = %+v; want b then a", c.MultiSeries)
	}
	renderToHTML(t, c)
}

func TestScatterChartRefusesWhatItCannotPair(t *testing.T) {
	quiet(t)
	xs := insyra.NewDataList(1, 2, 3)
	var typedNil *insyra.DataList
	tests := []struct {
		name   string
		series ScatterSeries
		want   []string
	}{
		{"nil X", ScatterSeries{Name: "s", X: typedNil, Y: xs}, []string{`"s"`, "X and Y"}},
		{"different lengths", ScatterSeries{Name: "s", X: xs, Y: insyra.NewDataList(1, 2)}, []string{`"s"`, "3 X values", "2 Y values"}},
		{"no points", ScatterSeries{Name: "s", X: insyra.NewDataList(), Y: insyra.NewDataList()}, []string{`"s"`, "no points"}},
		{"text in X", ScatterSeries{Name: "s", X: insyra.NewDataList(1, "two", 3), Y: xs}, []string{`"s"`, "X", "row index 1", "two"}},
		{"numeric string in Y", ScatterSeries{Name: "s", X: xs, Y: insyra.NewDataList(1, 2, "3")}, []string{`"s"`, "Y", "row index 2"}},
		{"nil cell in Y", ScatterSeries{Name: "s", X: xs, Y: insyra.NewDataList(nil, 2, 3)}, []string{`"s"`, "Y", "row index 0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insyra.ClearErrors()
			c, err := CreateScatterChart(ScatterChartConfig{}, tt.series)
			if err == nil || c != nil {
				t.Fatalf("chart %v, error %v; want a nil chart and an error", c, err)
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

func TestHeatmapChartIsTheSharedName(t *testing.T) {
	quiet(t)
	c, err := CreateHeatmapChart(HeatmapChartConfig{},
		NewHeatmapPoint("mon", "am", 1),
		NewHeatmapMissingPoint("tue", "pm"),
	)
	if err != nil {
		t.Fatal(err)
	}
	renderToHTML(t, c)

	// The former names are aliases and wrappers for one release.
	points := []HeatMapPoint[string, string]{NewHeatMapPoint("mon", "am", 1), NewHeatMapMissingPoint("tue", "pm")}
	old, err := CreateHeatMap(HeatMapConfig{}, points...)
	if err != nil || old == nil {
		t.Fatalf("CreateHeatMap = %v, %v", old, err)
	}
}

func TestRadarChartTakesSeriesVariadically(t *testing.T) {
	quiet(t)
	cfg := RadarChartConfig{Indicators: []string{"x", "y"}}
	c, err := CreateRadarChart(cfg,
		RadarSeries{Name: "a", Values: []float32{1, 2}},
		RadarSeries{Name: "b", Values: []float32{2, 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	html := renderToHTML(t, c)
	for _, name := range []string{`"a"`, `"b"`} {
		if !strings.Contains(html, name) {
			t.Errorf("the rendered radar does not name series %s", name)
		}
	}
}

func TestBarAndLineRefuseAnEmptyList(t *testing.T) {
	quiet(t)
	calls := map[string]func() (bool, error){
		"CreateBarChart": func() (bool, error) {
			c, err := CreateBarChart(BarChartConfig{}, nil, insyra.NewDataList(1, 2), insyra.NewDataList())
			return c == nil, err
		},
		"CreateLineChart": func() (bool, error) {
			c, err := CreateLineChart(LineChartConfig{}, nil, insyra.NewDataList(1, 2), insyra.NewDataList())
			return c == nil, err
		},
	}
	for fn, call := range calls {
		t.Run(fn, func(t *testing.T) {
			insyra.ClearErrors()
			isNil, err := call()
			if err == nil || !isNil {
				t.Fatalf("nil chart %v, error %v; want a nil chart and an error", isNil, err)
			}
			if !strings.Contains(err.Error(), "list 3") || !strings.Contains(err.Error(), "empty") {
				t.Errorf("error %q does not name list 3 as empty", err)
			}
			if recs := insyra.GetAllErrors(); len(recs) != 0 {
				t.Errorf("the failure was also logged: %+v", recs)
			}
		})
	}
}

func TestPositionsCountTheCallersArguments(t *testing.T) {
	quiet(t)
	insyra.ClearErrors()
	_, err := CreateBarChart(BarChartConfig{}, nil, insyra.NewDataList(1.0, math.NaN()))
	if err == nil || !strings.Contains(err.Error(), "list 2") {
		t.Fatalf("error %v; want it to name list 2, the NaN's argument", err)
	}

	insyra.ClearErrors()
	if _, err := CreateLineChart(LineChartConfig{}, nil, insyra.NewDataList(1, 2)); err != nil {
		t.Fatal(err)
	}
	if !loggedWarning("CreateLineChart", "data list 1 is nil") {
		t.Errorf("no warning naming data list 1: %+v", insyra.GetAllErrors())
	}
}
