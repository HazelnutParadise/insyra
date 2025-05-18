package insyra

import (
	"fmt"
	"math"
	"strings"
)

// Summary displays a comprehensive statistical summary of the DataList directly to the console.
func (dl *DataList) Summary() {
	defer func() {
		dl.mu.Unlock()
		go reorganizeMemory(dl)
	}()
	dl.mu.Lock()

	// Get terminal window width
	width := getDataListTerminalWidth()

	// Generate data title
	dataTitle := "DataList Statistical Summary"
	if dl.name != "" {
		dataTitle += ": " + dl.name
	}
	// Calculate all statistics first, then display them at once
	// Basic statistical data structure
	type StatisticsData struct {
		// Basic information
		totalCount     int
		numericCount   int
		numericPercent float64

		// Central tendency
		mean       float64
		median     float64
		modeValues []float64

		// Dispersion
		min      float64
		max      float64
		rangeVal float64
		stdev    float64
		variance float64

		// Quantiles
		q1  float64
		q3  float64
		iqr float64
	}
	// Check if DataList is empty
	totalCount := len(dl.data)
	if totalCount == 0 {
		// Display header - using yellow as DataList primary color
		fmt.Println(ColorText("1;33", dataTitle))
		fmt.Println(strings.Repeat("=", min(width, 80)))
		fmt.Println(ColorText("3;33", "Empty dataset"))
		return
	}

	// Count numeric values
	numericCount := 0
	for _, v := range dl.data {
		if isNumeric(v) {
			numericCount++
		}
	}

	// Start displaying results
	// Display header - using green as DataList primary color
	fmt.Println(ColorText("1;33", dataTitle))
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Initialize statistics
	stats := StatisticsData{
		totalCount:     totalCount,
		numericCount:   numericCount,
		numericPercent: float64(numericCount) / float64(totalCount) * 100,
	}

	// Display basic info
	fmt.Printf("Total items: %s\n", ColorText("1;33", fmt.Sprintf("%d", stats.totalCount)))
	fmt.Printf("Numeric items: %s\n", ColorText("1;33", fmt.Sprintf("%d (%.1f%%)", stats.numericCount, stats.numericPercent)))
	fmt.Println()

	// Calculate statistics only if we have numeric values
	if numericCount > 0 {
		// Calculate all statistical indicators
		stats.mean = dl.Mean()
		stats.median = dl.Median()
		stats.min = dl.Min()
		stats.max = dl.Max()
		stats.rangeVal = dl.Range()
		stats.stdev = dl.Stdev()
		stats.variance = dl.Var()
		stats.q1 = dl.Quartile(1)
		stats.q3 = dl.Quartile(3)
		stats.iqr = dl.IQR()
		stats.modeValues = dl.Mode()
	} // Calculate statistics only if we have numeric values
	if numericCount == 0 {
		fmt.Println(ColorText("3;33", "No numeric data available for statistical analysis"))
		return
	}

	// Create a nice table with borders for statistics
	fmt.Println(ColorText("1;33", "Statistical Summary"))

	// 使用新的自適應表格顯示邏輯
	displayDataListSummaryTable(stats, width)
}

// formatFloat 用於格式化浮點數，處理 NaN 值
func formatFloat(f float64) string {
	if math.IsNaN(f) {
		return "N/A"
	}
	return fmt.Sprintf("%.4f", f)
}

// displayDataListSummaryTable 以更靈活的方式顯示DataList摘要統計表格
// 可以根據終端寬度自動調整顯示格式，避免表格歪斜
func displayDataListSummaryTable(stats struct {
	// Basic information
	totalCount     int
	numericCount   int
	numericPercent float64

	// Central tendency
	mean       float64
	median     float64
	modeValues []float64

	// Dispersion
	min      float64
	max      float64
	rangeVal float64
	stdev    float64
	variance float64

	// Quantiles
	q1  float64
	q3  float64
	iqr float64
}, width int) {
	// 計算表格顯示寬度
	statNameWidth := 16
	valueWidth := 15

	// 根據終端寬度動態調整表格列寬
	if width < 60 {
		totalWidth := max(40, width-6) // 保留最小總寬度
		statNameWidth = max(12, totalWidth*6/10)
		valueWidth = totalWidth - statNameWidth
	}

	// 建立表格格式
	headerFmt := "│ %-" + fmt.Sprintf("%d", statNameWidth) + "s │ %-" + fmt.Sprintf("%d", valueWidth) + "s │\n"
	dividerLine := "├" + strings.Repeat("─", statNameWidth+2) + "┼" + strings.Repeat("─", valueWidth+2) + "┤"
	topLine := "┌" + strings.Repeat("─", statNameWidth+2) + "┬" + strings.Repeat("─", valueWidth+2) + "┐"
	bottomLine := "└" + strings.Repeat("─", statNameWidth+2) + "┴" + strings.Repeat("─", valueWidth+2) + "┘"

	// Central Tendency section
	fmt.Println(topLine)
	fmt.Printf(ColorText("1;32", headerFmt), "Central Tendency", "Value")
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Mean", formatFloat(stats.mean))
	fmt.Printf(headerFmt, "Median", formatFloat(stats.median))

	// Mode calculation
	modeStr := "N/A" // Default value if mode can't be calculated
	if len(stats.modeValues) == 1 {
		modeStr = formatFloat(stats.modeValues[0])
	} else if len(stats.modeValues) > 1 {
		modes := make([]string, 0, len(stats.modeValues))
		for _, v := range stats.modeValues {
			modes = append(modes, formatFloat(v))
		}

		// 如果模式值太長，進行特殊處理
		combined := strings.Join(modes, ", ")
		if len(combined) > valueWidth-2 {
			if len(modes) > 3 {
				// 只顯示前兩個值加...
				combined = modes[0] + ", " + modes[1] + ", ..."
			}
			// 即使這樣還是太長，就截斷
			if len(combined) > valueWidth-2 {
				combined = combined[:valueWidth-5] + "..."
			}
		}
		modeStr = combined
	}
	fmt.Printf(headerFmt, "Mode", modeStr)
	fmt.Println(dividerLine)

	// Dispersion section
	fmt.Printf(ColorText("1;32", headerFmt), "Dispersion", "Value")
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Minimum", formatFloat(stats.min))
	fmt.Printf(headerFmt, "Maximum", formatFloat(stats.max))
	fmt.Printf(headerFmt, "Range", formatFloat(stats.rangeVal))
	fmt.Printf(headerFmt, "Variance", formatFloat(stats.variance))
	fmt.Printf(headerFmt, "Std Deviation", formatFloat(stats.stdev))
	fmt.Println(dividerLine)

	// Quantiles section
	fmt.Printf(ColorText("1;32", headerFmt), "Quantiles", "Value")
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Q1 (25%)", formatFloat(stats.q1))
	fmt.Printf(headerFmt, "Q2 (50%)", formatFloat(stats.median))
	fmt.Printf(headerFmt, "Q3 (75%)", formatFloat(stats.q3))
	fmt.Printf(headerFmt, "IQR", formatFloat(stats.iqr))
	fmt.Println(bottomLine)
}
