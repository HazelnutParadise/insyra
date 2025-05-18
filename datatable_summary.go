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

	// 在持有鎖的情況下直接讀取 dt.name，不通過 GetName() 方法
	tableName := dt.name
	columns := dt.columns

	dt.mu.Unlock()

	// Get terminal width
	width := getTerminalWidth()

	// Generate table title
	tableTitle := "DataTable Statistical Summary"
	// 檢查 DataTable 的名稱，不使用 GetName()
	if tableName != "" {
		tableTitle += ": " + tableName
	}

	// Display header - 使用青色作為 DataTable 的主要顏色
	fmt.Println(ColorText("1;36", tableTitle))
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Check if DataTable is empty
	if len(columns) == 0 {
		fmt.Println(ColorText("3;33", "Empty dataset"))
		return
	}

	// Print column statistics
	for i, col := range columns {
		colIndex := generateColIndex(i)
		colName := col.name
		if colName != "" {
			colIndex = fmt.Sprintf("%s (%s)", colIndex, colName)
		}
		fmt.Printf("\n%s\n", ColorText("1;33", fmt.Sprintf("Column: %s", colIndex)))
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

	// 先進行所有計算，再一次性顯示結果
	// 基本統計數據結構
	type StatisticsData struct {
		// 基本信息
		totalCount     int
		numericCount   int
		numericPercent float64

		// 集中趨勢
		mean       float64
		median     float64
		modeValues []float64

		// 離散程度
		min      float64
		max      float64
		rangeVal float64
		stdev    float64
		variance float64

		// 分位數
		q1  float64
		q3  float64
		iqr float64
	}

	// 初始化統計數據
	stats := StatisticsData{
		totalCount:     totalCount,
		numericCount:   numericCount,
		numericPercent: float64(numericCount) / float64(totalCount) * 100,
	}

	// 檢查是否有數值數據
	if numericCount == 0 {
		// 顯示基本信息
		fmt.Printf("Total items: %s\n", ColorText("1;33", fmt.Sprintf("%d", stats.totalCount)))
		fmt.Printf("Numeric items: %s\n", ColorText("1;33", fmt.Sprintf("%d (%.1f%%)", stats.numericCount, stats.numericPercent)))
		fmt.Println(ColorText("3;33", "No numeric data available for statistical analysis"))
		return
	}

	// 計算所有統計指標
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

	// 顯示統計結果
	// 首先顯示基本信息
	fmt.Printf("Total items: %s\n", ColorText("1;33", fmt.Sprintf("%d", stats.totalCount)))
	fmt.Printf("Numeric items: %s\n", ColorText("1;33", fmt.Sprintf("%d (%.1f%%)", stats.numericCount, stats.numericPercent)))
	// 計算表格顯示寬度
	statNameWidth := 16 // 統計指標名稱欄位寬度
	valueWidth := 15    // 值欄位寬度

	// 創建帶邊框的統計表格
	headerFmt := "│ %-" + fmt.Sprintf("%d", statNameWidth) + "s │ %-" + fmt.Sprintf("%d", valueWidth) + "s │\n"
	dividerLine := "├" + strings.Repeat("─", statNameWidth+2) + "┼" + strings.Repeat("─", valueWidth+2) + "┤"
	topLine := "┌" + strings.Repeat("─", statNameWidth+2) + "┬" + strings.Repeat("─", valueWidth+2) + "┐"
	bottomLine := "└" + strings.Repeat("─", statNameWidth+2) + "┴" + strings.Repeat("─", valueWidth+2) + "┘"
	// 集中趨勢部分
	fmt.Println(topLine)
	fmt.Printf(headerFmt, "Central Tendency", "Value")
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Mean", formatFloat(stats.mean))
	fmt.Printf(headerFmt, "Median", formatFloat(stats.median))

	// 處理眾數
	if stats.modeValues != nil {
		modeStr := ""
		for i, mv := range stats.modeValues {
			if i > 0 {
				modeStr += ", "
			}
			modeStr += fmt.Sprintf("%v", mv)
		}
		fmt.Printf(headerFmt, "Mode", modeStr)
	}
	// 離散程度部分
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Dispersion", "Value")
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Minimum", formatFloat(stats.min))
	fmt.Printf(headerFmt, "Maximum", formatFloat(stats.max))
	fmt.Printf(headerFmt, "Range", formatFloat(stats.rangeVal))
	fmt.Printf(headerFmt, "Std Deviation", formatFloat(stats.stdev))
	fmt.Printf(headerFmt, "Variance", formatFloat(stats.variance))
	// 分位數部分
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Quantiles", "Value")
	fmt.Println(dividerLine)
	fmt.Printf(headerFmt, "Q1 (25%)", formatFloat(stats.q1))
	fmt.Printf(headerFmt, "Q2 (50%)", formatFloat(stats.median)) // Q2 與中位數相同
	fmt.Printf(headerFmt, "Q3 (75%)", formatFloat(stats.q3))
	fmt.Printf(headerFmt, "IQR", formatFloat(stats.iqr))
	fmt.Println(bottomLine)
}
