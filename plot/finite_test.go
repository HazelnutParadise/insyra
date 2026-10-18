package plot

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// #409: go-echarts drops the JSON error for NaN and an infinity, so the page
// came out blank with no error. Every constructor refuses such a value.
func TestConstructorsRefuseNonFiniteValues(t *testing.T) {
	quiet(t)
	nan, inf := math.NaN(), math.Inf(1)
	cases := map[string]func() error{
		"bar nan":    func() error { _, e := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, nan)); return e },
		"bar inf":    func() error { _, e := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, inf)); return e },
		"line nan":   func() error { _, e := CreateLineChart(LineChartConfig{}, insyra.NewDataList(nan, 1.0)); return e },
		"pie nan":    func() error { _, e := CreatePieChart(PieChartConfig{}, PieItem{Name: "a", Value: nan}); return e },
		"gauge nan":  func() error { _, e := CreateGaugeChart(GaugeChartConfig{}, nan); return e },
		"funnel inf": func() error { _, e := CreateFunnelChart(FunnelChartConfig{}, map[string]float64{"a": inf}); return e },
		"kline nan":  func() error { _, e := CreateKlineChart(KlineChartConfig{}, KlinePoint{Open: nan}); return e },
		"sankey nan": func() error {
			_, e := CreateSankeyChart(SankeyChartConfig{}, SankeyLink{Source: "a", Target: "b", Value: float32(nan)})
			return e
		},
		"scatter inf": func() error {
			_, e := CreateScatterChart(ScatterChartConfig{}, map[string][]ScatterPoint{"s": {{X: 1, Y: inf}}})
			return e
		},
		"river nan": func() error {
			_, e := CreateThemeRiverChart(ThemeRiverChartConfig{}, ThemeRiverData{Date: "2026/01/01", Value: nan, Name: "n"})
			return e
		},
		"radar nan": func() error {
			_, e := CreateRadarChart(RadarChartConfig{Indicators: []string{"a"}}, []RadarSeries{{Name: "s", Values: []float32{float32(nan)}}})
			return e
		},
		"box nan": func() error {
			_, e := CreateBoxPlot(BoxPlotConfig{}, BoxPlotSeries{Name: "s", Data: []insyra.IDataList{insyra.NewDataList(1.0, nan)}})
			return e
		},
		"box all text": func() error {
			_, e := CreateBoxPlot(BoxPlotConfig{}, BoxPlotSeries{Name: "s", Data: []insyra.IDataList{insyra.NewDataList("a", "b")}})
			return e
		},
	}
	for name, call := range cases {
		err := call()
		if err == nil || !strings.HasPrefix(err.Error(), "plot: ") {
			t.Errorf("%s: want a plot error, got %v", name, err)
		}
	}
}

func TestConstructorsStillDrawFiniteValues(t *testing.T) {
	quiet(t)
	if _, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1.0, 2.0)); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateGaugeChart(GaugeChartConfig{}, 3.5); err != nil {
		t.Fatal(err)
	}
}
