// plot/internal/colors.go

package internal

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
// Arrangement: Colors are interleaved so that index 0 is from family 1,
// index 1 is from family 2, index 2 is from family 3, etc.
// This maximizes visual distinction between consecutive series.
func DefaultColors() []string {
	return []string{
		// Round 1: First color from each family
		"#4A90E2", // Blues: Bright sky blue
		"#50C878", // Greens: Emerald
		"#9B59B6", // Purples: Amethyst
		"#E74C3C", // Reds/Pinks: Vibrant red
		"#F39C12", // Oranges/Ambers: Golden orange
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
