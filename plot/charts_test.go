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
		build func() (Renderable, error)
	}{
		{name: "bar", build: func() (Renderable, error) {
			return CreateBarChart(BarChartConfig{
				Title: "bar chart", XAxis: []string{"a", "b", "c"},
				XAxisName: "x", YAxisName: "y", ShowLabels: true,
			}, numbers("s", 1, 2, 3))
		}},
		{name: "line", build: func() (Renderable, error) {
			return CreateLineChart(LineChartConfig{
				Title: "line chart", Smooth: true, FillArea: true,
			}, numbers("s", 1, 2, 3), numbers("t", 3, 2, 1))
		}},
		{name: "scatter", build: func() (Renderable, error) {
			return CreateScatterChart(ScatterChartConfig{
				Title: "scatter chart", SymbolSize: 12, SplitLine: true,
			}, map[string][]ScatterPoint{
				"s": {{X: 0, Y: 1}, {X: 1, Y: 2}},
			})
		}},
		{name: "pie", build: func() (Renderable, error) {
			return CreatePieChart(PieChartConfig{
				Title: "pie chart", ShowLabels: true, ShowPercent: true,
			}, PieItem{Name: "a", Value: 1}, PieItem{Name: "b", Value: 2})
		}},
		{name: "boxplot", build: func() (Renderable, error) {
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
		{name: "heatmap", build: func() (Renderable, error) {
			return CreateHeatMap(HeatMapConfig{Title: "heatmap chart"},
				NewHeatMapPoint("mon", "am", 1),
				NewHeatMapPoint("tue", "pm", 2),
				NewHeatMapMissingPoint("wed", "am"),
			)
		}},
		{name: "radar", build: func() (Renderable, error) {
			return CreateRadarChart(RadarChartConfig{
				Title: "radar chart", Indicators: []string{"speed", "power"},
			}, []RadarSeries{{Name: "s", Values: []float32{1, 2}}})
		}},
		{name: "kline", build: func() (Renderable, error) {
			day := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
			return CreateKlineChart(KlineChartConfig{Title: "kline chart", DataZoom: true},
				KlinePoint{Date: day, Open: 1, High: 3, Low: 0.5, Close: 2},
				KlinePoint{Date: day.AddDate(0, 0, 1), Open: 2, High: 4, Low: 1.5, Close: 3},
			)
		}},
		{name: "funnel", build: func() (Renderable, error) {
			return CreateFunnelChart(FunnelChartConfig{Title: "funnel chart", ShowLabels: true},
				map[string]float64{"visit": 100, "buy": 10})
		}},
		{name: "gauge", build: func() (Renderable, error) {
			return CreateGaugeChart(GaugeChartConfig{Title: "gauge chart", SeriesName: "load"}, 42)
		}},
		{name: "sankey", build: func() (Renderable, error) {
			return CreateSankeyChart(SankeyChartConfig{Title: "sankey chart", ShowLabels: true},
				SankeyLink{Source: "a", Target: "b", Value: 1},
				SankeyLink{Source: "b", Target: "c", Value: 2},
			)
		}},
		{name: "themeriver", build: func() (Renderable, error) {
			return CreateThemeRiverChart(ThemeRiverChartConfig{Title: "themeriver chart"},
				ThemeRiverData{Date: "2026-09-11", Name: "a", Value: 1},
				ThemeRiverData{Date: "2026-09-12", Name: "a", Value: 2},
			)
		}},
		{name: "wordcloud", build: func() (Renderable, error) {
			return CreateWordCloud(WordCloudConfig{Title: "wordcloud chart", Shape: WordCloudShapeCircle},
				insyra.NewDataList(1, 2, 3).SetName("words"))
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chart, err := tt.build()
			if err != nil {
				t.Fatalf("building the chart: %v", err)
			}
			html := renderToHTML(t, chart)
			if !strings.Contains(html, tt.name+" chart") {
				t.Errorf("the rendered HTML does not carry the title")
			}
			if !strings.Contains(html, "echarts") {
				t.Errorf("the rendered HTML does not look like an ECharts page")
			}
		})
	}
}

func TestSaveHTML(t *testing.T) {
	quiet(t)
	chart, err := CreateBarChart(BarChartConfig{Title: "t"}, numbers("s", 1, 2, 3))
	if err != nil {
		t.Fatal(err)
	}

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

	chart, err := CreateBarChart(BarChartConfig{
		Title:    "<script>alert(1)</script>",
		Subtitle: "a & b",
	}, numbers("s", 1))
	if err != nil {
		t.Fatal(err)
	}
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

	chart, err := CreateBarChart(BarChartConfig{}, numbers("s", 1))
	if err != nil {
		t.Fatal(err)
	}
	if err := SavePNG(chart, filepath.Join(t.TempDir(), "chart.png"), true, true); err == nil {
		t.Fatal("SavePNG accepted two fallback arguments")
	}
}
