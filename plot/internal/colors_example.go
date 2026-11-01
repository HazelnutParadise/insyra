// plot/internal/colors_example.go

package internal

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/palette"
)

// PrintColorPalette prints the default palette in the order series use it.
func PrintColorPalette() {
	colors := DefaultColors()
	names := palette.Names()
	fmt.Println("Insyra chart palette (series 1 to 9, then it starts again):")
	for i, c := range colors {
		fmt.Printf("  [%d] %s  %s\n", i+1, c, names[i])
	}
}

// GenerateHTMLColorPreview returns an HTML page showing the default palette
// as swatches in the order series use it.
func GenerateHTMLColorPreview() string {
	colors := DefaultColors()
	names := palette.Names()
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Insyra Chart Palette</title>
<style>
body { font-family: sans-serif; background: #fff; color: #222; margin: 24px; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px; }
.swatch { height: 64px; border-radius: 4px; }
.label { font-size: 13px; margin-top: 4px; }
</style>
</head>
<body>
<h1>Insyra chart palette</h1>
<p>Series 1 to 9 take these colours in order; a tenth series starts again from the first.</p>
<div class="grid">
`)
	for i, c := range colors {
		fmt.Fprintf(&b, "<div><div class=\"swatch\" style=\"background:%s\"></div><div class=\"label\">%d. %s<br>%s</div></div>\n", c, i+1, names[i], c)
	}
	b.WriteString("</div>\n</body>\n</html>\n")
	return b.String()
}
