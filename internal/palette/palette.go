// Package palette holds the series colours plot and gplot share, so series i
// is the same colour in an interactive chart and a static one.
package palette

import (
	"image/color"
	"strconv"
)

// hex is the palette: nine colours taken from the Insyra website, in a fixed
// order checked for colour-blind safety between neighbours. Slots 1 and 2 are
// the brand blue #1F449B and orange #F9AC41 tuned for charts (lighter blue,
// deeper orange) so thin lines and small markers stay readable on white; the
// rest are the website's colours as they are. Slot 9 is a neutral gray for a
// series that folds the rest into "Other", which should not look like one
// more category. A tenth series starts again from the first colour rather
// than taking a new hue.
var hex = []string{Blue, Orange, Red, Teal, Purple, Pink, Green, BrightBlue, Gray}

// The palette's colours, and the brand's background colour. Cream is too pale
// to be a series, so it is not in the palette; it is for a chart's background.
const (
	Blue       = "#2550B0" // Insyra blue, tuned for charts
	Orange     = "#EE9A2C" // Insyra orange, tuned for charts
	Red        = "#D73E3E" // Insyra red
	Teal       = "#2A9D8F"
	Purple     = "#764BA2"
	Pink       = "#FF69B4"
	Green      = "#4CAF50"
	BrightBlue = "#2196F3"
	Gray       = "#7F8C8D" // For "Other"
	Cream      = "#FFF4E6"
)

// names are the colours' names, in palette order.
var names = []string{"Insyra blue", "Insyra orange", "Insyra red", "Teal", "Purple", "Pink", "Green", "Bright blue", "Gray"}

// Hex returns a copy of the palette as #RRGGBB strings.
func Hex() []string {
	return append([]string(nil), hex...)
}

// Names returns a copy of the colours' names, in palette order.
func Names() []string {
	return append([]string(nil), names...)
}

// Color returns colour i of the palette, wrapping past its end.
func Color(i int) color.Color {
	h := hex[i%len(hex)]
	v, _ := strconv.ParseUint(h[1:], 16, 32)
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}
