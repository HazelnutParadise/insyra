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

	// refused asserts a nil chart and an error: nothing was left to draw.
	refused := func(t *testing.T, isNil bool, err error) {
		t.Helper()
		if err == nil || !isNil {
			t.Errorf("got a chart (nil=%v) and error %v; want a nil chart and an error", isNil, err)
		}
	}
	// drawn asserts a chart and no error, and renders it.
	drawn := func(t *testing.T, c Renderable, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("the whole chart was refused because of one nil list: %v", err)
		}
		renderToHTML(t, c)
	}

	t.Run("bar, all nil", func(t *testing.T) {
		c, err := CreateBarChart(BarChartConfig{}, nil)
		refused(t, c == nil, err)
		c, err = CreateBarChart(BarChartConfig{}, typedNil)
		refused(t, c == nil, err)
	})
	t.Run("bar, one nil among real ones", func(t *testing.T) {
		c, err := CreateBarChart(BarChartConfig{Title: "t"}, nil, good)
		drawn(t, c, err)
	})

	t.Run("line, all nil", func(t *testing.T) {
		c, err := CreateLineChart(LineChartConfig{}, nil, typedNil)
		refused(t, c == nil, err)
	})
	t.Run("line, one nil among real ones", func(t *testing.T) {
		c, err := CreateLineChart(LineChartConfig{Title: "t"}, good, nil)
		drawn(t, c, err)
	})

	t.Run("wordcloud", func(t *testing.T) {
		c, err := CreateWordCloud(WordCloudConfig{}, nil)
		refused(t, c == nil, err)
		c, err = CreateWordCloud(WordCloudConfig{}, typedNil)
		refused(t, c == nil, err)
	})

	t.Run("boxplot, all nil", func(t *testing.T) {
		c, err := CreateBoxPlot(BoxPlotConfig{}, BoxPlotSeries{
			Name: "s",
			Data: []insyra.IDataList{nil, typedNil},
		})
		refused(t, c == nil, err)
	})
	t.Run("boxplot, one nil among real ones", func(t *testing.T) {
		c, err := CreateBoxPlot(BoxPlotConfig{Title: "t"}, BoxPlotSeries{
			Name: "s",
			Data: []insyra.IDataList{nil, insyra.NewDataList(1, 2, 3, 4, 5).SetName("g")},
		})
		drawn(t, c, err)
	})
}

// The snapshot dependency reads the image format from the extension and slices
// an empty string when there is none, so a path like "out" used to panic before
// it even looked for a browser.
func TestSavePNG_PathWithoutExtension(t *testing.T) {
	quiet(t)

	chart, err := CreateBarChart(BarChartConfig{}, insyra.NewDataList(1, 2).SetName("s"))
	if err != nil {
		t.Fatal(err)
	}
	err = SavePNG(chart, filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("SavePNG accepted a path with no extension")
	}
	if !strings.Contains(err.Error(), "extension") {
		t.Errorf("error %q does not mention the missing extension", err)
	}
}
