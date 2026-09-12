package plot

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// error-philosophy says no insyra package panics under the default config.
// Every chart here reads its data through IDataList.AtomicDo, which
// dereferences the receiver, so a nil list used to take the program down.

func TestNilDataListIsSkippedNotFatal(t *testing.T) {
	quiet(t)

	var typedNil *insyra.DataList
	good := insyra.NewDataList(1, 2, 3).SetName("s")

	t.Run("bar, all nil", func(t *testing.T) {
		if c := CreateBarChart(BarChartConfig{}, nil); c != nil {
			t.Error("returned a chart with nothing to draw")
		}
		if c := CreateBarChart(BarChartConfig{}, typedNil); c != nil {
			t.Error("returned a chart for a typed nil list")
		}
	})
	t.Run("bar, one nil among real ones", func(t *testing.T) {
		c := CreateBarChart(BarChartConfig{Title: "t"}, nil, good)
		if c == nil {
			t.Fatal("the whole chart was refused because of one nil list")
		}
		renderToHTML(t, c)
	})

	t.Run("line, all nil", func(t *testing.T) {
		if c := CreateLineChart(LineChartConfig{}, nil, typedNil); c != nil {
			t.Error("returned a chart with nothing to draw")
		}
	})
	t.Run("line, one nil among real ones", func(t *testing.T) {
		c := CreateLineChart(LineChartConfig{Title: "t"}, good, nil)
		if c == nil {
			t.Fatal("the whole chart was refused because of one nil list")
		}
		renderToHTML(t, c)
	})

	t.Run("wordcloud", func(t *testing.T) {
		if c := CreateWordCloud(WordCloudConfig{}, nil); c != nil {
			t.Error("returned a chart for a nil list")
		}
		if c := CreateWordCloud(WordCloudConfig{}, typedNil); c != nil {
			t.Error("returned a chart for a typed nil list")
		}
	})

	t.Run("boxplot, all nil", func(t *testing.T) {
		c := CreateBoxPlot(BoxPlotConfig{}, BoxPlotSeries{
			Name: "s",
			Data: []insyra.IDataList{nil, typedNil},
		})
		if c != nil {
			t.Error("returned a chart with nothing to draw")
		}
	})
	t.Run("boxplot, one nil among real ones", func(t *testing.T) {
		c := CreateBoxPlot(BoxPlotConfig{Title: "t"}, BoxPlotSeries{
			Name: "s",
			Data: []insyra.IDataList{nil, insyra.NewDataList(1, 2, 3, 4, 5).SetName("g")},
		})
		if c == nil {
			t.Fatal("the whole chart was refused because of one nil list")
		}
		renderToHTML(t, c)
	})
}

// The snapshot dependency reads the image format from the extension and slices
// an empty string when there is none, so a path like "out" used to panic before
// it even looked for a browser.
func TestSavePNG_PathWithoutExtension(t *testing.T) {
	quiet(t)

	chart := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1, 2).SetName("s"))
	err := SavePNG(chart, filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("SavePNG accepted a path with no extension")
	}
	if !strings.Contains(err.Error(), "extension") {
		t.Errorf("error %q does not mention the missing extension", err)
	}
}
