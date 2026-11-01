package gplot

import (
	"image/color"

	"github.com/HazelnutParadise/insyra/internal/palette"
)

// seriesColor is the colour of series i in every chart: the palette plot
// uses, which opens with Insyra's blue, orange and red, so a series is the
// same colour in an interactive chart and a static one.
func seriesColor(i int) color.Color {
	return palette.Color(i)
}
