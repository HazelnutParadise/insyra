// plot/internal/colors_example.go

package internal

import (
	"fmt"
)

// PrintColorPalette prints all colors in the default palette with their hex codes and descriptions.
// Colors are displayed in their usage order (interleaved), showing how they appear when used in charts.
func PrintColorPalette() {
	colors := DefaultColors()

	colorNames := []string{
		// Round 1
		"Bright sky blue", "Emerald", "Amethyst", "Vibrant red",
		"Golden orange", "Turquoise", "Slate gray", "Pumpkin",
		// Round 2
		"Soft azure", "Soft jade", "Lavender purple", "Rose pink",
		"Tangerine", "Bright cyan", "Mocha brown", "Dark turquoise",
		// Round 3
		"Muted periwinkle", "Fresh lime", "Orchid", "Coral red",
		"Soft amber", "Teal green", "Warm taupe", "Deep purple",
		// Round 4
		"Deep royal blue", "Sea green", "Royal purple", "Dusty rose",
		"Burnt orange", "Aquamarine", "Soft brown", "Rust",
		// Round 5
		"Light ocean blue", "Forest green", "Soft mauve", "Light red",
		"Light orange", "Seafoam", "Medium gray", "Kelly green",
		// Round 6
		"Teal blue", "Apple green", "Medium purple", "Berry red",
		"Mandarin", "Dark teal", "Caramel", "Brick red",
		// Round 7
		"Powder blue", "Meadow green", "Periwinkle purple", "Watermelon",
		"Honey gold", "Sky cyan", "Blue gray", "Ocean blue",
		// Round 8
		"Medium cobalt", "Sage green", "Light violet", "Salmon pink",
		"Marigold", "Ocean teal", "Cocoa", "Sunny yellow",
	}

	colorFamilies := []string{
		"Blues", "Greens", "Purples", "Reds/Pinks",
		"Oranges/Ambers", "Teals/Cyans", "Warm Neutrals", "Cool Accents",
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║   Insyra Plot Default Color Palette (64 Colors - Interleaved)     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Colors are arranged in interleaved order for maximum visual distinction")
	fmt.Println("between consecutive data series. Each round cycles through all 8 families.")
	fmt.Println()

	// Display by rounds
	for round := 0; round < 8; round++ {
		fmt.Printf("┌─ Round %d (Colors %d-%d)\n", round+1, round*8+1, (round+1)*8)
		fmt.Println("│")

		for i := 0; i < 8; i++ {
			idx := round*8 + i
			family := colorFamilies[i]
			prefix := "├──"
			if i == 7 {
				prefix = "└──"
			}
			fmt.Printf("%s [%2d] %s  %-18s  (%s)\n", prefix, idx+1, colors[idx], colorNames[idx], family)
		}
		fmt.Println()
	}

	fmt.Println("Usage Examples:")
	fmt.Println("  • 3 series: Blue, Green, Purple (maximum distinction)")
	fmt.Println("  • 8 series: All 8 families used once (optimal)")
	fmt.Println("  • 16 series: All 8 families used twice (still clear)")
	fmt.Println()
	fmt.Println("API:")
	fmt.Println("  • Colors auto-apply when not specified in chart config")
	fmt.Println("  • internal.GetColor(index) - get specific color")
	fmt.Println("  • internal.GetColors(n) - get n colors (cycles if n > 64)")
	fmt.Println()
}

// PrintColorsByFamily prints colors organized by their color families.
// Useful for understanding the color system structure.
func PrintColorsByFamily() {
	colors := DefaultColors()

	families := []struct {
		Name        string
		Description string
		Indices     []int
	}{
		{
			"Blues", "Professional and trustworthy",
			[]int{0, 8, 16, 24, 32, 40, 48, 56},
		},
		{
			"Greens", "Fresh and harmonious",
			[]int{1, 9, 17, 25, 33, 41, 49, 57},
		},
		{
			"Purples", "Creative and elegant",
			[]int{2, 10, 18, 26, 34, 42, 50, 58},
		},
		{
			"Reds/Pinks", "Energetic and warm",
			[]int{3, 11, 19, 27, 35, 43, 51, 59},
		},
		{
			"Oranges/Ambers", "Enthusiastic and confident",
			[]int{4, 12, 20, 28, 36, 44, 52, 60},
		},
		{
			"Teals/Cyans", "Balanced and refreshing",
			[]int{5, 13, 21, 29, 37, 45, 53, 61},
		},
		{
			"Warm Neutrals", "Sophisticated and elegant",
			[]int{6, 14, 22, 30, 38, 46, 54, 62},
		},
		{
			"Cool Accents", "Diverse and interesting",
			[]int{7, 15, 23, 31, 39, 47, 55, 63},
		},
	}

	colorNames := [][]string{
		{"Bright sky blue", "Soft azure", "Muted periwinkle", "Deep royal blue",
			"Light ocean blue", "Teal blue", "Powder blue", "Medium cobalt"},
		{"Emerald", "Soft jade", "Fresh lime", "Sea green",
			"Forest green", "Apple green", "Meadow green", "Sage green"},
		{"Amethyst", "Lavender purple", "Orchid", "Royal purple",
			"Soft mauve", "Medium purple", "Periwinkle purple", "Light violet"},
		{"Vibrant red", "Rose pink", "Coral red", "Dusty rose",
			"Light red", "Berry red", "Watermelon", "Salmon pink"},
		{"Golden orange", "Tangerine", "Soft amber", "Burnt orange",
			"Light orange", "Mandarin", "Honey gold", "Marigold"},
		{"Turquoise", "Bright cyan", "Teal green", "Aquamarine",
			"Seafoam", "Dark teal", "Sky cyan", "Ocean teal"},
		{"Slate gray", "Mocha brown", "Warm taupe", "Soft brown",
			"Medium gray", "Caramel", "Blue gray", "Cocoa"},
		{"Pumpkin", "Dark turquoise", "Deep purple", "Rust",
			"Kelly green", "Brick red", "Ocean blue", "Sunny yellow"},
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║             Insyra Plot Colors - Organized by Family              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	for familyIdx, family := range families {
		fmt.Printf("┌─ %d. %s - %s\n", familyIdx+1, family.Name, family.Description)
		fmt.Println("│   (Positions in palette)")
		fmt.Println("│")

		for i, idx := range family.Indices {
			colorName := colorNames[familyIdx][i]
			prefix := "├──"
			if i == len(family.Indices)-1 {
				prefix = "└──"
			}
			fmt.Printf("%s [Pos %2d] %s  →  %s\n", prefix, idx+1, colors[idx], colorName)
		}
		fmt.Println()
	}
}

// GenerateHTMLColorPreview generates an HTML file content that displays all colors
// in a visually appealing grid format showing the interleaved arrangement.
func GenerateHTMLColorPreview() string {
	colors := DefaultColors()

	html := `<!DOCTYPE html>
<html lang="zh-TW">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Insyra Plot Color Palette</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 40px 20px;
            min-height: 100vh;
        }
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #4A90E2 0%, #50C878 100%);
            color: white;
            padding: 60px 40px;
            text-align: center;
        }
        .header h1 {
            font-size: 3em;
            margin-bottom: 10px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
        }
        .header p {
            font-size: 1.2em;
            opacity: 0.95;
        }
        .info-section {
            background: #f8f9fa;
            padding: 30px 40px;
            border-bottom: 1px solid #e0e0e0;
        }
        .info-section h2 {
            color: #333;
            margin-bottom: 15px;
            font-size: 1.5em;
        }
        .info-section p {
            color: #666;
            line-height: 1.8;
            margin-bottom: 10px;
        }
        .content {
            padding: 40px;
        }
        .round {
            margin-bottom: 40px;
        }
        .round-header {
            margin-bottom: 20px;
            padding: 15px 20px;
            background: linear-gradient(135deg, #f5f5f5 0%, #e8e8e8 100%);
            border-radius: 10px;
            border-left: 4px solid #4A90E2;
        }
        .round-title {
            font-size: 1.5em;
            color: #333;
            margin-bottom: 5px;
        }
        .round-description {
            color: #666;
            font-size: 1em;
        }
        .color-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
            gap: 15px;
        }
        .color-card {
            background: white;
            border-radius: 12px;
            overflow: hidden;
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
            transition: transform 0.3s ease, box-shadow 0.3s ease;
            cursor: pointer;
            position: relative;
        }
        .color-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 8px 24px rgba(0,0,0,0.15);
        }
        .color-swatch {
            height: 100px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            color: white;
            text-shadow: 1px 1px 2px rgba(0,0,0,0.3);
            font-size: 1.3em;
        }
        .color-info {
            padding: 12px;
            background: #f9f9f9;
        }
        .color-hex {
            font-family: 'Courier New', monospace;
            font-size: 1em;
            color: #333;
            font-weight: bold;
            margin-bottom: 4px;
        }
        .color-name {
            font-size: 0.85em;
            color: #666;
            margin-bottom: 4px;
        }
        .color-family {
            font-size: 0.75em;
            color: #999;
            font-style: italic;
        }
        .copied-tooltip {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: rgba(0,0,0,0.8);
            color: white;
            padding: 8px 16px;
            border-radius: 6px;
            font-size: 0.9em;
            opacity: 0;
            pointer-events: none;
            transition: opacity 0.3s;
        }
        .color-card.copied .copied-tooltip {
            opacity: 1;
        }
        .footer {
            text-align: center;
            padding: 30px;
            background: #f5f5f5;
            color: #666;
        }
        @media (max-width: 768px) {
            .header h1 {
                font-size: 2em;
            }
            .color-grid {
                grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
                gap: 12px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎨 Insyra Plot Color Palette</h1>
            <p>64 精心設計的交錯排列配色方案</p>
        </div>

        <div class="info-section">
            <h2>🌈 智能配色系統</h2>
            <p><strong>交錯排列設計：</strong>不同色系的顏色交錯排列，確保相鄰數據系列具有最大的視覺差異。</p>
            <p><strong>使用順序：</strong>第1-8個數據系列會分別使用藍、綠、紫、紅、橙、青、灰、彩八種不同色系，一目了然。</p>
            <p><strong>自動應用：</strong>在 plot 包中創建圖表時，如果不指定顏色，系統會自動按順序應用此配色。</p>
        </div>

        <div class="content">
`

	colorNames := []string{
		// Round 1
		"Bright sky blue", "Emerald", "Amethyst", "Vibrant red",
		"Golden orange", "Turquoise", "Slate gray", "Pumpkin",
		// Round 2
		"Soft azure", "Soft jade", "Lavender purple", "Rose pink",
		"Tangerine", "Bright cyan", "Mocha brown", "Dark turquoise",
		// Round 3
		"Muted periwinkle", "Fresh lime", "Orchid", "Coral red",
		"Soft amber", "Teal green", "Warm taupe", "Deep purple",
		// Round 4
		"Deep royal blue", "Sea green", "Royal purple", "Dusty rose",
		"Burnt orange", "Aquamarine", "Soft brown", "Rust",
		// Round 5
		"Light ocean blue", "Forest green", "Soft mauve", "Light red",
		"Light orange", "Seafoam", "Medium gray", "Kelly green",
		// Round 6
		"Teal blue", "Apple green", "Medium purple", "Berry red",
		"Mandarin", "Dark teal", "Caramel", "Brick red",
		// Round 7
		"Powder blue", "Meadow green", "Periwinkle purple", "Watermelon",
		"Honey gold", "Sky cyan", "Blue gray", "Ocean blue",
		// Round 8
		"Medium cobalt", "Sage green", "Light violet", "Salmon pink",
		"Marigold", "Ocean teal", "Cocoa", "Sunny yellow",
	}

	colorFamilies := []string{
		"Blues", "Greens", "Purples", "Reds/Pinks",
		"Oranges/Ambers", "Teals/Cyans", "Warm Neutrals", "Cool Accents",
	}

	for round := 0; round < 8; round++ {
		html += fmt.Sprintf(`
            <div class="round">
                <div class="round-header">
                    <div class="round-title">Round %d (第 %d 輪)</div>
                    <div class="round-description">Colors %d-%d - 八種色系各一色，最大化視覺差異</div>
                </div>
                <div class="color-grid">
`, round+1, round+1, round*8+1, (round+1)*8)

		for i := 0; i < 8; i++ {
			idx := round*8 + i
			colorHex := colors[idx]
			colorName := colorNames[idx]
			family := colorFamilies[i]

			html += fmt.Sprintf(`
                    <div class="color-card" onclick="copyColor(this, '%s')">
                        <div class="color-swatch" style="background-color: %s;">
                            #%d
                        </div>
                        <div class="color-info">
                            <div class="color-hex">%s</div>
                            <div class="color-name">%s</div>
                            <div class="color-family">%s</div>
                        </div>
                        <div class="copied-tooltip">已複製!</div>
                    </div>
`, colorHex, colorHex, idx+1, colorHex, colorName, family)
		}

		html += `
                </div>
            </div>
`
	}

	html += `
        </div>

        <div class="footer">
            <p><strong>點擊任意顏色卡片複製色碼到剪貼板</strong></p>
            <p style="margin-top: 10px; font-size: 0.9em;">Insyra Plot Internal Color Palette © 2024</p>
            <p style="margin-top: 5px; font-size: 0.85em; color: #999;">
                交錯排列設計 · 最佳視覺體驗 · 永不過時
            </p>
        </div>
    </div>

    <script>
        function copyColor(element, colorCode) {
            navigator.clipboard.writeText(colorCode).then(() => {
                element.classList.add('copied');
                setTimeout(() => {
                    element.classList.remove('copied');
                }, 1500);
            });
        }
    </script>
</body>
</html>`

	return html
}
