package plot

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot/internal"
)

// The constants are the palette, in order, so a config can name them.
func TestColorConstantsAreThePalette(t *testing.T) {
	want := []string{ColorBlue, ColorOrange, ColorRed, ColorTeal, ColorPurple, ColorPink, ColorGreen, ColorBrightBlue, ColorGray}
	got := internal.DefaultColors()
	if len(got) != len(want) {
		t.Fatalf("%d colours in the palette, %d constants", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("series %d is %s; the constant says %s", i+1, got[i], want[i])
		}
	}
}

func TestConstantsReachTheChart(t *testing.T) {
	quiet(t)
	c, err := CreateBarChart(BarChartConfig{BackgroundColor: ColorCream, Colors: []string{ColorBlue, ColorGray}},
		insyra.NewDataList(1, 2).SetName("a"), insyra.NewDataList(2, 1).SetName("Other"))
	if err != nil {
		t.Fatal(err)
	}
	html := renderToHTML(t, c)
	for _, want := range []string{ColorCream, ColorBlue, ColorGray} {
		if !strings.Contains(html, want) {
			t.Errorf("the page does not use %s", want)
		}
	}
}
