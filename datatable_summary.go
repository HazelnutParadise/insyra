package insyra

import (
	"fmt"
	"strings"
)

// Summary displays a comprehensive statistical summary of the DataTable directly to the console.
// It shows various descriptive statistics for each column including count, mean, median, min, max,
// standard deviation, and other metrics when applicable.
// The output is formatted for easy reading with proper color coding.
func (dt *DataTable) Summary() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Get terminal width
	width := getTerminalWidth()

	// Generate table title
	tableTitle := "DataTable Statistical Summary"
	// 檢查 DataTable 的名稱訪問方法
	// 嘗試使用不同的訪問方法
	// if dt.name != "" { // 使用直接訪問欄位的方式
	// 	tableTitle += ": " + dt.name
	// }

	// Display header - 使用青色作為 DataTable 的主要顏色
	fmt.Println(ColorText("1;36", tableTitle))
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Check if DataTable is empty
	if len(dt.columns) == 0 {
		fmt.Println(ColorText("3;33", "Empty dataset"))
		return
	}

	// Print column statistics
	for i, col := range dt.columns {
		colIndex := generateColIndex(i)
		colName := col.name
		if colName != "" {
			colIndex = fmt.Sprintf("%s (%s)", colIndex, colName)
		}

		fmt.Printf("\n%s\n", ColorText("1;36", fmt.Sprintf("Column: %s", colIndex)))
		fmt.Println(strings.Repeat("-", min(width, 50)))

		showColumnStatistics(col.data)
	}
}

func showColumnStatistics(data []any) {
	// Create DataList from the column data
	dl := NewDataList()
	dl.Append(data...)

	// Count total and numeric items
	totalCount := len(data)
	numericCount := 0
	for _, v := range data {
		if isNumeric(v) {
			numericCount++
		}
	}

	// Display basic info
	fmt.Printf("Total items: %s\n", ColorText("1;36", fmt.Sprintf("%d", totalCount)))
	fmt.Printf("Numeric items: %s\n", ColorText("1;34", fmt.Sprintf("%d (%.1f%%)", numericCount, float64(numericCount)/float64(totalCount)*100)))

	// 檢查是否有數值數據
	if numericCount == 0 {
		fmt.Println(ColorText("3;33", "No numeric data available for statistical analysis"))
		return
	}

	// Calculate statistics
	mean := dl.Mean()
	median := dl.Median()
	min := dl.Min()
	max := dl.Max()
	rangeVal := dl.Range()
	stdev := dl.Stdev()
	variance := dl.Var()

	// Calculate quartiles
	q1 := dl.Quartile(1)
	q3 := dl.Quartile(3)
	iqr := dl.IQR()

	// Calculate table display widths
	statNameWidth := 16
	valueWidth := 15

	// Create a nice table with borders for statistics
	// Print table header
	headerFmt := "│ %-" + fmt.Sprintf("%d", statNameWidth) + "s │ %-" + fmt.Sprintf("%d", valueWidth) + "s │\n"
	dividerLine := "├" + strings.Repeat("─", statNameWidth+2) + "┼" + strings.Repeat("─", valueWidth+2) + "┤"
	topLine := "┌" + strings.Repeat("─", statNameWidth+2) + "┬" + strings.Repeat("─", valueWidth+2) + "┐"
	bottomLine := "└" + strings.Repeat("─", statNameWidth+2) + "┴" + strings.Repeat("─", valueWidth+2) + "┘"

	// Central Tendency section
	fmt.Println(topLine)
	fmt.Printf(headerFmt, ColorText("1;34", "Central Tendency"), ColorText("1;34", "Value"))
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Mean", formatFloat(mean))
	fmt.Printf(headerFmt, "Median", formatFloat(median))

	// Mode calculation
	modeValues := dl.Mode()
	if modeValues != nil {
		modeStr := ""
		for i, mv := range modeValues {
			if i > 0 {
				modeStr += ", "
			}
			modeStr += fmt.Sprintf("%v", mv)
		}
		fmt.Printf(headerFmt, "Mode", modeStr)
	}

	// Dispersion section
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, ColorText("1;34", "Dispersion"), ColorText("1;34", "Value"))
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Minimum", formatFloat(min))
	fmt.Printf(headerFmt, "Maximum", formatFloat(max))
	fmt.Printf(headerFmt, "Range", formatFloat(rangeVal))
	fmt.Printf(headerFmt, "Std Deviation", formatFloat(stdev))
	fmt.Printf(headerFmt, "Variance", formatFloat(variance))

	// Quantiles section
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, ColorText("1;34", "Quantiles"), ColorText("1;34", "Value"))
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Q1 (25%)", formatFloat(q1))
	fmt.Printf(headerFmt, "Q2 (50%)", formatFloat(median)) // Q2 is the same as median
	fmt.Printf(headerFmt, "Q3 (75%)", formatFloat(q3))
	fmt.Printf(headerFmt, "IQR", formatFloat(iqr))
	fmt.Println(bottomLine)
}
