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
var hex = []string{
	"#2550B0", // Insyra blue, tuned for charts
	"#EE9A2C", // Insyra orange, tuned for charts
	"#D73E3E", // Insyra red
	"#2A9D8F", // Teal
	"#764BA2", // Purple
	"#FF69B4", // Pink
	"#4CAF50", // Green
	"#2196F3", // Bright blue
	"#7F8C8D", // Gray, for "Other"
}

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
