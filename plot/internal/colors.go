// plot/internal/colors.go

package internal

import "github.com/HazelnutParadise/insyra/internal/palette"

// DefaultColors returns the series palette plot and gplot share: nine colours
// from the Insyra website, opening with its blue, orange and red and ending
// with a neutral gray for "Other". The order is fixed and checked for
// colour-blind safety between neighbours; a tenth series starts again from the
// first colour. The palette lives in internal/palette.
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
// If n exceeds the palette size, colors repeat from the first.
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
