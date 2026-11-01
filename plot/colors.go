package plot

import "github.com/HazelnutParadise/insyra/internal/palette"

// The chart palette and the brand's background colours, usable in a config's
// Colors and BackgroundColor fields. They are plain strings, so they mix with
// your own values. Without Colors, series take the first nine in this order;
// ColorGray is for a series that folds the rest into "Other". ColorCream and
// ColorWhite are too pale to be series and are meant for BackgroundColor.
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
	ColorWhite      = palette.White
)
