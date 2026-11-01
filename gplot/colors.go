package gplot

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/palette"
	"golang.org/x/image/colornames"
	"gonum.org/v1/plot"
)

// The chart palette and the brand's background colour, usable in a config's
// Colors and BackgroundColor fields. They are plain strings, so they mix with
// your own values. Without Colors, series take the first nine in this order;
// ColorGray is for a series that folds the rest into "Other". ColorCream is too
// pale to be a series and is meant for BackgroundColor.
const (
	ColorBlue       = palette.Blue
	ColorOrange     = palette.Orange
	ColorRed        = palette.Red
	ColorTeal       = palette.Teal
	ColorPurple     = palette.Purple
	ColorPink       = palette.Pink
	ColorGreen      = palette.Green
	ColorBrightBlue = palette.BrightBlue
	ColorGray       = palette.Gray
	ColorCream      = palette.Cream
)

// seriesColor is the default colour of series i in every chart: the palette
// plot uses, which opens with Insyra's blue, orange and red, so a series is the
// same colour in an interactive chart and a static one.
func seriesColor(i int) color.Color {
	return palette.Color(i)
}

// seriesColors is what a config's Colors field reads as: the colours to use in
// place of the palette, repeated when there are more series than colours, the
// way plot's are. Empty means the palette.
type seriesColors []color.Color

// at returns the colour of series i.
func (c seriesColors) at(i int) color.Color {
	if len(c) == 0 {
		return seriesColor(i)
	}
	return c[i%len(c)]
}

// readColors parses a config's Colors field, or says which value it cannot read.
func readColors(funcName string, values []string) (seriesColors, error) {
	colors := make(seriesColors, len(values))
	for i, v := range values {
		c, ok := parseColor(v)
		if !ok {
			return nil, chartError(funcName, "Colors[%d] is %q; use #RGB, #RRGGBB, #RRGGBBAA or a CSS colour name", i, v)
		}
		colors[i] = c
	}
	return colors, nil
}

// newPlot starts a chart with the background a config asks for; empty keeps
// gonum's white.
func newPlot(funcName, background string) (*plot.Plot, error) {
	plt := plot.New()
	if background == "" {
		return plt, nil
	}
	c, ok := parseColor(background)
	if !ok {
		return nil, chartError(funcName, "BackgroundColor is %q; use #RGB, #RRGGBB, #RRGGBBAA or a CSS colour name", background)
	}
	plt.BackgroundColor = c
	return plt, nil
}

// parseColor reads #RGB, #RRGGBB, #RRGGBBAA or a CSS colour name in any case.
func parseColor(s string) (color.Color, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !strings.HasPrefix(s, "#") {
		c, ok := colornames.Map[s]
		return c, ok
	}
	hex := s[1:]
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) == 6 {
		hex += "ff"
	}
	if len(hex) != 8 {
		return nil, false
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, false
	}
	// A #RRGGBBAA value is straight alpha, which is what color.NRGBA holds.
	return color.NRGBA{R: uint8(v >> 24), G: uint8(v >> 16), B: uint8(v >> 8), A: uint8(v)}, true
}
