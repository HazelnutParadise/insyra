package plot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// The whole package had no test file. Every chart constructor and both save
// paths are exercised here. SavePNG is deliberately only tested for its
// argument check: rendering a PNG launches a real Chrome, which CI does not
// have, and its online fallback would upload the chart data.

// quiet silences the warnings these tests provoke and puts the global config
// back afterwards.
func quiet(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	panicOnError := insyra.Config.GetPanicOnError()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	insyra.Config.SetPanicOnError(false)
	t.Cleanup(func() {
		insyra.Config.SetLogLevel(level)
		insyra.Config.SetPanicOnError(panicOnError)
	})
}

// renderToHTML writes the chart and returns what landed on disk. A chart that
// builds but cannot render is not a working chart.
func renderToHTML(t *testing.T, chart Renderable) string {
	t.Helper()
	if chart == nil {
		t.Fatal("the chart is nil")
	}
	path := filepath.Join(t.TempDir(), "chart.html")
	if err := SaveHTML(chart, path); err != nil {
		t.Fatalf("SaveHTML: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("SaveHTML wrote an empty file")
	}
	return string(b)
}

func numbers(name string, vals ...any) insyra.IDataList {
	return insyra.NewDataList(vals...).SetName(name)
}

// Every constructor, built with a title and rendered to HTML. The title has to
// survive into the output, which is the cheapest proof that the chart was
// actually configured and not just allocated.
func TestEveryChartBuildsAndRenders(t *testing.T) {
	quiet(t)

	tests := []struct {
		name  string
		build func() Renderable
	}{
		{name: "bar", build: func() Renderable {
			return CreateBarChart(BarChartConfig{
				Title: "bar chart", XAxis: []string{"a", "b", "c"},
				XAxisName: "x", YAxisName: "y", ShowLabels: true,
			}, numbers("s", 1, 2, 3))
		}},
		{name: "line", build: func() Renderable {
			return CreateLineChart(LineChartConfig{
				Title: "line chart", Smooth: true, FillArea: true,
			}, numbers("s", 1, 2, 3), numbers("t", 3, 2, 1))
		}},
		{name: "scatter", build: func() Renderable {
			return CreateScatterChart(ScatterChartConfig{
				Title: "scatter chart", SymbolSize: 12, SplitLine: true,
			}, map[string][]ScatterPoint{
				"s": {{X: 0, Y: 1}, {X: 1, Y: 2}},
			})
		}},
		{name: "pie", build: func() Renderable {
			return CreatePieChart(PieChartConfig{
				Title: "pie chart", ShowLabels: true, ShowPercent: true,
			}, PieItem{Name: "a", Value: 1}, PieItem{Name: "b", Value: 2})
		}},
		{name: "boxplot", build: func() Renderable {
			return CreateBoxPlot(BoxPlotConfig{
				Title: "boxplot chart", XAxis: []string{"g1", "g2"},
			}, BoxPlotSeries{
				Name: "s",
				Data: []insyra.IDataList{
					numbers("g1", 1, 2, 3, 4, 5),
					numbers("g2", 2, 3, 4, 5, 6),
				},
			})
		}},
		{name: "heatmap", build: func() Renderable {
			return CreateHeatMap(HeatMapConfig{Title: "heatmap chart"},
				HeatMapPoint("mon", "am", 1),
				HeatMapPoint("tue", "pm", 2),
				HeatMapMissingPoint("wed", "am"),
			)
		}},
		{name: "radar", build: func() Renderable {
			return CreateRadarChart(RadarChartConfig{
				Title: "radar chart", Indicators: []string{"speed", "power"},
			}, []RadarSeries{{Name: "s", Values: []float32{1, 2}}})
		}},
		{name: "kline", build: func() Renderable {
			day := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
			return CreateKlineChart(KlineChartConfig{Title: "kline chart", DataZoom: true},
				KlinePoint{Date: day, Open: 1, High: 3, Low: 0.5, Close: 2},
				KlinePoint{Date: day.AddDate(0, 0, 1), Open: 2, High: 4, Low: 1.5, Close: 3},
			)
		}},
		{name: "funnel", build: func() Renderable {
			return CreateFunnelChart(FunnelChartConfig{Title: "funnel chart", ShowLabels: true},
				map[string]float64{"visit": 100, "buy": 10})
		}},
		{name: "gauge", build: func() Renderable {
			return CreateGaugeChart(GaugeChartConfig{Title: "gauge chart", SeriesName: "load"}, 42)
		}},
		{name: "sankey", build: func() Renderable {
			return CreateSankeyChart(SankeyChartConfig{Title: "sankey chart", ShowLabels: true},
				SankeyLink{Source: "a", Target: "b", Value: 1},
				SankeyLink{Source: "b", Target: "c", Value: 2},
			)
		}},
		{name: "themeriver", build: func() Renderable {
			return CreateThemeRiverChart(ThemeRiverChartConfig{Title: "themeriver chart"},
				ThemeRiverData{Date: "2026-09-11", Name: "a", Value: 1},
				ThemeRiverData{Date: "2026-09-12", Name: "a", Value: 2},
			)
		}},
		{name: "wordcloud", build: func() Renderable {
			return CreateWordCloud(WordCloudConfig{Title: "wordcloud chart", Shape: WordCloudShapeCircle},
				insyra.NewDataList(1, 2, 3).SetName("words"))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := renderToHTML(t, tt.build())
			if !strings.Contains(html, tt.name+" chart") {
				t.Errorf("the rendered HTML does not carry the title")
			}
			if !strings.Contains(html, "echarts") {
				t.Errorf("the rendered HTML does not look like an ECharts page")
			}
		})
	}
}

// Every constructor that documents a nil return for empty input.
func TestEmptyInputGivesNoChart(t *testing.T) {
	quiet(t)

	if c := CreateBarChart(BarChartConfig{}); c != nil {
		t.Error("CreateBarChart with no data returned a chart")
	}
	if c := CreateLineChart(LineChartConfig{}); c != nil {
		t.Error("CreateLineChart with no data returned a chart")
	}
	if c := CreateScatterChart(ScatterChartConfig{}, map[string][]ScatterPoint{}); c != nil {
		t.Error("CreateScatterChart with an empty map returned a chart")
	}
	if c := CreatePieChart(PieChartConfig{}); c != nil {
		t.Error("CreatePieChart with no items returned a chart")
	}
	if c := CreateBoxPlot(BoxPlotConfig{}); c != nil {
		t.Error("CreateBoxPlot with no series returned a chart")
	}
	if c := CreateHeatMap[string, string](HeatMapConfig{}); c != nil {
		t.Error("CreateHeatMap with no points returned a chart")
	}
	if c := CreateRadarChart(RadarChartConfig{}, nil); c != nil {
		t.Error("CreateRadarChart with no series returned a chart")
	}
	// A radar chart with series but nothing to measure them against.
	if c := CreateRadarChart(RadarChartConfig{}, []RadarSeries{{Name: "s", Values: []float32{1}}}); c != nil {
		t.Error("CreateRadarChart with neither indicators nor maximums returned a chart")
	}
	if c := CreateKlineChart(KlineChartConfig{}); c != nil {
		t.Error("CreateKlineChart with no points returned a chart")
	}
	if c := CreateFunnelChart(FunnelChartConfig{}, map[string]float64{}); c != nil {
		t.Error("CreateFunnelChart with an empty map returned a chart")
	}
	if c := CreateSankeyChart(SankeyChartConfig{}); c != nil {
		t.Error("CreateSankeyChart with no links returned a chart")
	}
	if c := CreateThemeRiverChart(ThemeRiverChartConfig{}); c != nil {
		t.Error("CreateThemeRiverChart with no data returned a chart")
	}
	if c := CreateWordCloud(WordCloudConfig{}, insyra.NewDataList()); c != nil {
		t.Error("CreateWordCloud with an empty list returned a chart")
	}
	// A gauge always has a value, so it never refuses.
	if c := CreateGaugeChart(GaugeChartConfig{}, 0); c == nil {
		t.Error("CreateGaugeChart returned nil")
	}
}

// A calendar heat map needs time-valued x points and calendar options; without
// either it refuses rather than rendering something meaningless.
func TestCreateHeatMap_Calendar(t *testing.T) {
	quiet(t)

	if c := CreateHeatMap(HeatMapConfig{UseCalendar: true},
		HeatMapPoint("mon", "am", 1)); c != nil {
		t.Error("a calendar heat map with string x values returned a chart")
	}

	day := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	if c := CreateHeatMap(HeatMapConfig{UseCalendar: true},
		HeatMapPoint(day, "am", 1)); c != nil {
		t.Error("a calendar heat map with no calendar options returned a chart")
	}
}

func TestSaveHTML(t *testing.T) {
	quiet(t)
	chart := CreateBarChart(BarChartConfig{Title: "t"}, numbers("s", 1, 2, 3))

	t.Run("animation off", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "chart.html")
		if err := SaveHTML(chart, path, false); err != nil {
			t.Fatalf("SaveHTML: %v", err)
		}
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("nothing useful was written: %v", err)
		}
	})

	t.Run("unwritable path", func(t *testing.T) {
		err := SaveHTML(chart, filepath.Join(t.TempDir(), "no", "such", "dir", "chart.html"))
		if err == nil {
			t.Fatal("SaveHTML returned no error for an unwritable path")
		}
		if !strings.Contains(err.Error(), "chart.html") {
			t.Errorf("error %q does not name the file", err)
		}
	})
}

// The title and subtitle are HTML-escaped, so a chart built from user data
// cannot inject markup into the page.
func TestSaveHTML_EscapesTitles(t *testing.T) {
	quiet(t)

	chart := CreateBarChart(BarChartConfig{
		Title:    "<script>alert(1)</script>",
		Subtitle: "a & b",
	}, numbers("s", 1))
	html := renderToHTML(t, chart)

	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Error("the title was written into the page as markup")
	}
	if !strings.Contains(html, "&amp;") {
		t.Error("the subtitle's ampersand was not escaped")
	}
}

// SavePNG needs a local Chrome, so only its argument check is tested here.
// Passing a second value would be the caller asking for two different fallback
// policies at once.
func TestSavePNG_RejectsExtraArguments(t *testing.T) {
	quiet(t)

	chart := CreateBarChart(BarChartConfig{}, numbers("s", 1))
	err := SavePNG(chart, filepath.Join(t.TempDir(), "chart.png"), true, true)
	if err == nil {
		t.Fatal("SavePNG accepted two fallback arguments")
	}
}
