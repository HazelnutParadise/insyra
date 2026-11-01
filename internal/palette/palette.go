// Package palette holds the series colours plot and gplot share, so series i
// is the same colour in an interactive chart and a static one.
package palette

import (
	"image/color"
	"strconv"
)

// hex is the palette: 64 colours in 8 families, interleaved so consecutive
// series differ. The first three are Insyra's brand colours, the ones the
// website uses: blue #1F449B, orange #F9AC41 and red #D73E3E.
var hex = []string{
	// Round 1: Insyra's three brand colours first, then the other families
	"#1F449B", // Blues: Insyra blue
	"#F9AC41", // Oranges/Ambers: Insyra orange
	"#D73E3E", // Reds/Pinks: Insyra red
	"#50C878", // Greens: Emerald
	"#9B59B6", // Purples: Amethyst
	"#1ABC9C", // Teals/Cyans: Turquoise
	"#95A5A6", // Warm Neutrals: Slate gray
	"#E67E22", // Cool Accents: Pumpkin

	// Round 2: Second color from each family
	"#5B9BD5", // Blues: Soft azure
	"#6DB784", // Greens: Soft jade
	"#8E7AB5", // Purples: Lavender purple
	"#F06292", // Reds/Pinks: Rose pink
	"#FF9F4A", // Oranges/Ambers: Tangerine
	"#26C6DA", // Teals/Cyans: Bright cyan
	"#8D6E63", // Warm Neutrals: Mocha brown
	"#16A085", // Cool Accents: Dark turquoise

	// Round 3: Third color from each family
	"#6B7FBD", // Blues: Muted periwinkle
	"#8BC34A", // Greens: Fresh lime
	"#B47EBF", // Purples: Orchid
	"#EC6B7A", // Reds/Pinks: Coral red
	"#FFB74D", // Oranges/Ambers: Soft amber
	"#4DB6AC", // Teals/Cyans: Teal green
	"#B8956A", // Warm Neutrals: Warm taupe
	"#9C27B0", // Cool Accents: Deep purple

	// Round 4: Fourth color from each family
	"#3D5A98", // Blues: Deep royal blue
	"#45B592", // Greens: Sea green
	"#7B68AB", // Purples: Royal purple
	"#D47A8A", // Reds/Pinks: Dusty rose
	"#E89944", // Oranges/Ambers: Burnt orange
	"#48C9B0", // Teals/Cyans: Aquamarine
	"#A1887F", // Warm Neutrals: Soft brown
	"#D35400", // Cool Accents: Rust

	// Round 5: Fifth color from each family
	"#5DADE2", // Blues: Light ocean blue
	"#66A266", // Greens: Forest green
	"#A67CB5", // Purples: Soft mauve
	"#E57373", // Reds/Pinks: Light red
	"#FFA726", // Oranges/Ambers: Light orange
	"#5CC5C0", // Teals/Cyans: Seafoam
	"#9E9E9E", // Warm Neutrals: Medium gray
	"#27AE60", // Cool Accents: Kelly green

	// Round 6: Sixth color from each family
	"#2E7D9A", // Blues: Teal blue
	"#7CB342", // Greens: Apple green
	"#9575CD", // Purples: Medium purple
	"#C7556B", // Reds/Pinks: Berry red
	"#FF8C42", // Oranges/Ambers: Mandarin
	"#43A19E", // Teals/Cyans: Dark teal
	"#AB8F7D", // Warm Neutrals: Caramel
	"#C0392B", // Cool Accents: Brick red

	// Round 7: Seventh color from each family
	"#7BA3CC", // Blues: Powder blue
	"#5FAD56", // Greens: Meadow green
	"#8B7FB8", // Purples: Periwinkle purple
	"#E85D75", // Reds/Pinks: Watermelon
	"#F8B739", // Oranges/Ambers: Honey gold
	"#50BCD4", // Teals/Cyans: Sky cyan
	"#90A4AE", // Warm Neutrals: Blue gray
	"#2980B9", // Cool Accents: Ocean blue

	// Round 8: Eighth color from each family
	"#4169A8", // Blues: Medium cobalt
	"#4D9D7C", // Greens: Sage green
	"#BA68C8", // Purples: Light violet
	"#D66B75", // Reds/Pinks: Salmon pink
	"#E6A23C", // Oranges/Ambers: Marigold
	"#3EACB4", // Teals/Cyans: Ocean teal
	"#7D6C65", // Warm Neutrals: Cocoa
	"#F1C40F", // Cool Accents: Sunny yellow
}

// Hex returns a copy of the palette as #RRGGBB strings.
func Hex() []string {
	return append([]string(nil), hex...)
}

// Color returns colour i of the palette, wrapping past its end.
func Color(i int) color.Color {
	h := hex[i%len(hex)]
	v, _ := strconv.ParseUint(h[1:], 16, 32)
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}
