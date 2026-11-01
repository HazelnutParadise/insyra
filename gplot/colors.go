package gplot

import (
	"image/color"

	"gonum.org/v1/plot/plotutil"
)

// Every chart colours series i from one palette, gonum's paired soft and dark
// colours, which share their order of hues: series 1 is red in every chart.
// A filled shape takes the soft colour; a line or a marker takes the dark one,
// which stays readable at a line's width.

// seriesFill is the colour of series i's filled shapes, such as bars.
func seriesFill(i int) color.Color {
	return plotutil.SoftColors[i%len(plotutil.SoftColors)]
}

// seriesStroke is the colour of series i's lines and markers.
func seriesStroke(i int) color.Color {
	return plotutil.DarkColors[i%len(plotutil.DarkColors)]
}
