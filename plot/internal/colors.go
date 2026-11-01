// plot/internal/colors.go

package internal

import "github.com/HazelnutParadise/insyra/internal/palette"

// DefaultColors returns a carefully curated palette of 64 colors designed for data visualization.
// This palette is designed to be:
// - Comfortable and easy on the eyes
// - Aesthetically pleasing and timeless
// - Well-balanced across the color spectrum
// - Suitable for both light and dark backgrounds
// - Accessible with good contrast
// - Professional and modern
//
// The colors are arranged in an interleaved pattern across 8 color families.
// This ensures that consecutive colors in the palette are visually distinct,
// making it easy to differentiate between adjacent data series in charts.
//
// Color families:
// 1. Blues (trust, stability, professionalism)
// 2. Greens (growth, harmony, nature)
// 3. Purples (creativity, luxury, wisdom)
// 4. Reds/Pinks (energy, passion, warmth)
// 5. Oranges/Ambers (enthusiasm, confidence, vitality)
// 6. Teals/Cyans (balance, clarity, freshness)
// 7. Warm Neutrals (sophistication, elegance)
// 8. Cool Accents (diversity, interest, depth)
//
// Arrangement: Colors are interleaved across the families, so consecutive
// series differ. The first round opens with Insyra's three brand colours,
// blue, orange and red, the ones the website uses. The palette lives in
// internal/palette, which gplot shares.
func DefaultColors() []string {
	return palette.Hex()
}

// GetColor returns the color at the specified index from the default palette.
// If the index exceeds the palette size, it wraps around using modulo operation.
func GetColor(index int) string {
	colors := DefaultColors()
	return colors[index%len(colors)]
}

// GetColors returns a slice of n colors from the default palette.
// If n exceeds the palette size, colors will repeat in a cyclic manner.
func GetColors(n int) []string {
	if n <= 0 {
		return []string{}
	}

	colors := DefaultColors()
	result := make([]string, n)

	for i := 0; i < n; i++ {
		result[i] = colors[i%len(colors)]
	}

	return result
}
