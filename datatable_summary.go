package insyra

import (
	"fmt"
	"strings"
)

// Summary displays a comprehensive statistical summary of the DataTable directly to the console.
// It shows overall statistics for the entire table, including the number of rows and columns,
// and aggregate information about data types and values across the whole table.
// The output is formatted for easy reading with proper color coding.
func (dt *DataTable) Summary() {
	var tableName string
	var columns []*DataList
	var rowCount int

	dt.AtomicDo(func(dt *DataTable) {
		// Access dt.name directly while holding the lock, not through GetName() method
		tableName = dt.name
		columns = make([]*DataList, len(dt.columns))
		copy(columns, dt.columns)
		rowCount = dt.getMaxColLength()
	})

	// Get terminal width
	width := getTerminalWidth()

	// Generate table title
	tableTitle := "DataTable Statistical Summary" // Check DataTable name, not using GetName()
	if tableName != "" {
		tableTitle += ": " + tableName
	}

	// Display header - using blue as DataTable's primary color
	fmt.Println(colorText("1;36", tableTitle))
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Check if DataTable is empty
	if len(columns) == 0 {
		fmt.Println(colorText("2;37", "Empty dataset"))
		return
	} // Display overall table information
	fmt.Printf("Dimensions: %s rows × %s columns\n",
		colorText("1;36", fmt.Sprintf("%d", rowCount)),
		colorText("1;36", fmt.Sprintf("%d", len(columns))))
	// Count data types across the entire table
	var totalElements int
	var numericCount, stringCount, boolCount, otherCount int
	var numericValues []any // 儲存可轉換為數字的值

	for _, col := range columns {
		for _, v := range col.data {
			totalElements++
			if v == nil {
				otherCount++
				continue
			}

			switch v.(type) {
			case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
				numericCount++
				numericValues = append(numericValues, v)
			case string:
				stringCount++
			case bool:
				boolCount++
			default:
				otherCount++
			}
		}
	}

	// Display table-wide statistics (使用固定寬度格式，避免歪斜)
	fmt.Println("\n" + colorText("1;36", "Table-wide Statistics"))
	fmt.Println(strings.Repeat("-", min(width, 50)))

	fmt.Printf("Total elements: %s\n",
		colorText("1;36", fmt.Sprintf("%d", totalElements))) // 顯示各類型資料的比例
	if totalElements > 0 {
		fmt.Printf("Data types: %s numeric (%.1f%%), %s string (%.1f%%), %s boolean (%.1f%%), %s other (%.1f%%)\n",
			colorText("1;36", fmt.Sprintf("%d", numericCount)),
			float64(numericCount)/float64(totalElements)*100,
			colorText("1;36", fmt.Sprintf("%d", stringCount)),
			float64(stringCount)/float64(totalElements)*100,
			colorText("1;36", fmt.Sprintf("%d", boolCount)),
			float64(boolCount)/float64(totalElements)*100,
			colorText("1;36", fmt.Sprintf("%d", otherCount)),
			float64(otherCount)/float64(totalElements)*100)
	}

	// 只有當有數值型數據時才顯示統計資訊
	if len(numericValues) > 0 {
		// 創建臨時 DataList 用於計算統計值
		tempDl := NewDataList()
		tempDl.Append(numericValues...)

		// 計算並顯示表格範圍的數值統計
		statNameWidth := 16
		valueWidth := 15

		// 創建表格
		headerFmt := "│ %-" + fmt.Sprintf("%d", statNameWidth) + "s │ %-" + fmt.Sprintf("%d", valueWidth) + "s │\n"
		dividerLine := "├" + strings.Repeat("─", statNameWidth+2) + "┼" + strings.Repeat("─", valueWidth+2) + "┤"
		topLine := "┌" + strings.Repeat("─", statNameWidth+2) + "┬" + strings.Repeat("─", valueWidth+2) + "┐"
		bottomLine := "└" + strings.Repeat("─", statNameWidth+2) + "┴" + strings.Repeat("─", valueWidth+2) + "┘"
		fmt.Println("\n" + colorText("1;36", "Numeric Data Statistics (Across All Columns)"))
		fmt.Println(topLine)
		fmt.Printf(colorText("1;32", headerFmt), "Statistic", "Value")
		fmt.Println(dividerLine)
		fmt.Printf(headerFmt, "Count", fmt.Sprintf("%d", len(numericValues)))
		fmt.Printf(headerFmt, "Mean", formatFloat(tempDl.Mean()))
		fmt.Printf(headerFmt, "Median", formatFloat(tempDl.Median()))
		fmt.Printf(headerFmt, "Min", formatFloat(tempDl.Min()))
		fmt.Printf(headerFmt, "Max", formatFloat(tempDl.Max()))
		fmt.Printf(headerFmt, "Std Deviation", formatFloat(tempDl.Stdev()))
		fmt.Println(bottomLine)
	} // Display column information
	fmt.Println("\n" + colorText("1;36", "Column Overview"))
	fmt.Println(strings.Repeat("-", min(width, 50)))

	// 使用新的自適應表格顯示邏輯
	displayColumnOverviewTable(columns, width)
}

// getColumnQuickInfo returns data type information and quick statistics about a column
func getColumnQuickInfo(data []any, maxWidth ...int) (string, string) {
	if len(data) == 0 {
		return "Empty", "No data"
	}

	// 確定最大寬度
	statMaxWidth := 32
	if len(maxWidth) > 0 && maxWidth[0] > 0 {
		statMaxWidth = maxWidth[0]
	}

	// Count data types
	numericCount := 0
	stringCount := 0
	boolCount := 0
	otherCount := 0

	for _, v := range data {
		switch v.(type) {
		case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
			numericCount++
		case string:
			stringCount++
		case bool:
			boolCount++
		default:
			otherCount++
		}
	}

	// Determine predominant type
	dataType := "Mixed"
	if numericCount == len(data) {
		dataType = "Numeric"
	} else if stringCount == len(data) {
		dataType = "String"
	} else if boolCount == len(data) {
		dataType = "Boolean"
	}
	// Create quick stats
	var quickStats string
	if numericCount > 0 && numericCount == len(data) {
		// For numeric columns, show min, max, mean
		dl := NewDataList()
		dl.Append(data...)
		min, max, mean := dl.Min(), dl.Max(), dl.Mean()

		// 如果空間有限，使用更短的格式
		if statMaxWidth < 25 {
			quickStats = fmt.Sprintf("Min/Max/Mean: %.1f/%.1f/%.1f", min, max, mean)
		} else {
			quickStats = fmt.Sprintf("Min: %.2f, Max: %.2f, Mean: %.2f", min, max, mean)
		}
	} else if stringCount > 0 { // For string columns, show unique count
		uniqueMap := make(map[string]bool)
		for _, v := range data {
			if str, ok := v.(string); ok {
				uniqueMap[str] = true
			}
		}

		// 根據空間調整格式
		if statMaxWidth < 20 {
			quickStats = fmt.Sprintf("%d itm, %d uniq", len(data), len(uniqueMap))
		} else {
			quickStats = fmt.Sprintf("%d items, %d unique values", len(data), len(uniqueMap))
		}
	} else {
		// For mixed columns
		if statMaxWidth < 20 {
			// 極簡格式
			quickStats = fmt.Sprintf("%d itm (%d/%d/%d)",
				len(data), numericCount, stringCount, boolCount+otherCount)
		} else {
			quickStats = fmt.Sprintf("%d items (%d num, %d str, %d other)",
				len(data), numericCount, stringCount, boolCount+otherCount)
		}
	}
	// 縮短數據如果太長
	if len(quickStats) > statMaxWidth {
		quickStats = quickStats[:statMaxWidth-3] + "..."
	}

	return dataType, quickStats
}

// displayColumnOverviewTable 以更靈活的方式顯示列概覽
// 可以根據終端寬度自動調整顯示格式，避免表格歪斜
func displayColumnOverviewTable(columns []*DataList, width int) {
	if len(columns) == 0 {
		return
	}

	// 動態計算最佳列寬
	colNameWidth := 24
	typeWidth := 15
	statWidth := 32

	// 確保最小寬度和比例
	totalWidth := max(50, width-8)

	// 根據終端寬度調整列寬
	if width < 80 {
		colNameWidth = max(12, totalWidth*3/10)
		typeWidth = max(8, totalWidth*2/10)
		statWidth = totalWidth - colNameWidth - typeWidth
	}

	// 創建表格格式化字符串
	headerFmt := "│ %-" + fmt.Sprintf("%d", colNameWidth) + "s │ %-" + fmt.Sprintf("%d", typeWidth) + "s │ %-" + fmt.Sprintf("%d", statWidth) + "s │\n"
	dividerLine := "├" + strings.Repeat("─", colNameWidth+2) + "┼" + strings.Repeat("─", typeWidth+2) + "┼" + strings.Repeat("─", statWidth+2) + "┤"
	topLine := "┌" + strings.Repeat("─", colNameWidth+2) + "┬" + strings.Repeat("─", typeWidth+2) + "┬" + strings.Repeat("─", statWidth+2) + "┐"
	bottomLine := "└" + strings.Repeat("─", colNameWidth+2) + "┴" + strings.Repeat("─", typeWidth+2) + "┴" + strings.Repeat("─", statWidth+2) + "┘"

	// 打印表頭
	fmt.Println(topLine)
	fmt.Printf(colorText("1;32", headerFmt), "Column Name", "Data Type", "Quick Statistics")
	fmt.Println(dividerLine)

	// 為每一列顯示基本信息
	for i, col := range columns {
		colIndex := generateColIndex(i)
		colName := col.name
		if colName != "" {
			colIndex = fmt.Sprintf("%s (%s)", colIndex, colName)
			// 如果列名太長，進行截斷
			if len(colIndex) > colNameWidth-2 {
				colIndex = colIndex[:colNameWidth-5] + "..."
			}
		}

		// 獲取列數據類型和快速統計信息
		dataType, quickStats := getColumnQuickInfo(col.data, statWidth-2)

		// 確保數據不會破壞表格結構
		if len(dataType) > typeWidth {
			dataType = dataType[:typeWidth-3] + "..."
		}

		// 分行處理長統計數據，避免破壞表格結構
		if len(quickStats) > statWidth {
			// 將長數據分成多行顯示
			lines := splitStringByWidth(quickStats, statWidth-2)

			// 打印第一行數據
			fmt.Printf(headerFmt, colIndex, dataType, lines[0])

			// 如果有多行，繼續打印
			for j := 1; j < len(lines); j++ {
				fmt.Printf("│ %-"+fmt.Sprintf("%d", colNameWidth)+"s │ %-"+fmt.Sprintf("%d", typeWidth)+"s │ %-"+fmt.Sprintf("%d", statWidth)+"s │\n",
					"", "", lines[j])
			}

		} else {
			fmt.Printf(headerFmt, colIndex, dataType, quickStats)
		}

		if i < len(columns)-1 {
			fmt.Println(dividerLine)
		}
	}

	fmt.Println(bottomLine)
}

// splitStringByWidth 將字符串按照指定寬度分割成多行
func splitStringByWidth(s string, width int) []string {
	if width <= 0 || len(s) <= width {
		return []string{s}
	}

	var lines []string
	runes := []rune(s)
	length := len(runes)

	for i := 0; i < length; i += width {
		end := i + width
		if end > length {
			end = length
		}
		lines = append(lines, string(runes[i:end]))
	}

	return lines
}
