package insyra

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"
)

// ======================== DataTable ========================

func (dt *DataTable) Show() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 構建資料地圖，但不使用 Data() 方法以避免死鎖
	dataMap := make(map[string][]any)
	for i, col := range dt.columns {
		key := generateColIndex(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// 取得所有的列索引並排序
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)

	// 獲取終端視窗寬度
	width := getTerminalWidth()
	// 生成資料表標題
	tableTitle := "DataTable"

	// 在同一個鎖內獲取行數和列數，避免再次調用 Size() 導致死鎖
	rowCount := dt.getMaxColLength()
	colCount := len(dt.columns)
	tableSummary := fmt.Sprintf("(%d rows x %d columns)", rowCount, colCount)

	// 資料表基本信息的顯示 - 青色是 DataTable 的主要顏色
	fmt.Printf("\033[1;36m%s\033[0m %s\n", tableTitle, tableSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// 資料表為空的處理
	if rowCount == 0 || colCount == 0 {
		fmt.Println("\033[3;33m(empty)\033[0m")
		return
	}
	// 計算每一列的最大寬度
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := FormatValue(value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
		// 限制列寬度不超過特定值
		if colWidths[colIndex] > 30 {
			colWidths[colIndex] = 30
		}
	}

	// 計算 RowNames 的最大寬度，並顯示 RowIndex
	rowNames := make([]string, dt.getMaxColLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // 如果沒有名字則顯示為空
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // 加上 RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// 限制行名寬度不超過特定值
	if maxRowNameWidth > 25 {
		maxRowNameWidth = 25
	}
	// 嘗試顯示一些基本統計資訊
	if rowCount > 0 && colCount > 0 {
		// 為每一列顯示基本統計資訊
		hasNumbers := false
		statsInfo := "\033[3;36m stat: "

		for _, colIndex := range colIndices[:min(3, len(colIndices))] { // 只顯示前三列的統計資訊
			// 嘗試將列轉換為數值以計算統計數據
			dl := NewDataList(dataMap[colIndex])
			dl.ParseNumbers() // 嘗試解析為數字

			mean, colMin, colMax := dl.Mean(), dl.Min(), dl.Max()
			if !math.IsNaN(mean) && !math.IsNaN(colMin) && !math.IsNaN(colMax) {
				hasNumbers = true
				shortColName := strings.Split(colIndex, "(")[0] // 僅使用簡短的列名
				statsInfo += fmt.Sprintf("%s(mean=%.4g, range=[%.4g, %.4g]) ",
					shortColName, mean, colMin, colMax)
			}
		}

		if hasNumbers {
			// 如果有數值型列，則顯示統計資訊
			statsInfo += "\033[0m"
			fmt.Println(statsInfo)
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}

	// 根據當前視窗寬度動態調整要顯示的列數
	totalColsToShow := determineColumnsToShow(colIndices, colWidths, maxRowNameWidth, width)

	// 如果列數超過可顯示範圍，分頁顯示
	pageSize := totalColsToShow
	if pageSize <= 0 {
		pageSize = 1 // 至少顯示一列
	}

	// 計算需要多少頁
	totalPages := (len(colIndices) + pageSize - 1) / pageSize

	for page := 0; page < totalPages; page++ {
		start := page * pageSize
		end := (page + 1) * pageSize
		if end > len(colIndices) {
			end = len(colIndices)
		}

		currentPageCols := colIndices[start:end]

		if page > 0 {
			fmt.Println("\n\033[1;35m--- 繼續顯示 ---\033[0m")
		}

		if totalPages > 1 {
			pageInfo := fmt.Sprintf("--- 頁數 %d/%d ---", page+1, totalPages)
			fmt.Printf("\033[1;33m%s\033[0m\n", pageInfo)

			// 顯示頁面導航提示
			if page > 0 && page < totalPages-1 {
				fmt.Println("(查看更多請滾動屏幕)")
			}
		}
		// 打印列名
		fmt.Printf("\033[1;32m%-*s\033[0m", maxRowNameWidth+2, "RowNames")
		for _, colIndex := range currentPageCols {
			fmt.Printf(" \033[1;32m%-*s\033[0m", colWidths[colIndex]+1, TruncateString(colIndex, colWidths[colIndex]))
		}
		fmt.Println()

		// 打印分隔線
		printSeparator(maxRowNameWidth+2, currentPageCols, colWidths)

		// 打印行資料
		rowCount := dt.getMaxColLength()
		// 如果行數太多，只顯示前 20 行，中間省略，再顯示後 5 行
		if rowCount > 25 {
			// 顯示前 20 行
			printRowsColored(dataMap, 0, 20, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// 顯示省略號
			fmt.Printf("\033[1;33m%-*s\033[0m", maxRowNameWidth+2, "...")
			for range currentPageCols {
				fmt.Printf(" \033[1;33m%-*s\033[0m", colWidths[currentPageCols[0]]+1, "...")
			}
			fmt.Println()

			// 顯示後 5 行
			printRowsColored(dataMap, rowCount-5, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// 顯示資料摘要
			fmt.Printf("\n\033[3;36m共 %d 行資料，顯示了前 20 行和後 5 行\033[0m\n", rowCount)
		} else {
			// 行數不多，全部顯示
			printRowsColored(dataMap, 0, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)
		}

		// 如果是多頁，顯示页尾分隔线
		if totalPages > 1 {
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}
}

// 打印指定範圍的行（帶顏色）
func printRowsColored(dataMap map[string][]any, start, end int, rowNames []string, maxRowNameWidth int, colIndices []string, colWidths map[string]int) {
	for rowIndex := start; rowIndex < end; rowIndex++ {
		rowName := ""
		if rowIndex < len(rowNames) {
			rowName = rowNames[rowIndex]
		}
		// 使用淺灰色顯示行名
		fmt.Printf("\033[1;37m%-*s\033[0m", maxRowNameWidth+2, TruncateString(rowName, maxRowNameWidth))

		for _, colIndex := range colIndices {
			value := "nil"
			var rawValue any = nil
			if rowIndex < len(dataMap[colIndex]) {
				rawValue = dataMap[colIndex][rowIndex]
				if rawValue != nil {
					value = FormatValue(rawValue)
				}
			}

			// 根據值的類型使用不同的顏色
			valueColor := "\033[0m" // 默認顏色

			// 如果是空值，使用灰色
			if rawValue == nil {
				valueColor = "\033[2;37m" // 灰色
			} else {
				switch rawValue.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColor = "\033[0;34m" // 藍色表示數字
				case string:
					valueColor = "\033[0;32m" // 綠色表示字符串
				case bool:
					valueColor = "\033[0;35m" // 紫色表示布爾值
				}
			}

			fmt.Printf(" %s%-*s\033[0m", valueColor, colWidths[colIndex]+1, TruncateString(value, colWidths[colIndex]))
		}
		fmt.Println()
	}
}

// 打印分隔線
func printSeparator(rowNameWidth int, colIndices []string, colWidths map[string]int) {
	fmt.Print(strings.Repeat("-", rowNameWidth))
	for _, colIndex := range colIndices {
		fmt.Print(" " + strings.Repeat("-", colWidths[colIndex]+1))
	}
	fmt.Println()
}

// 獲取終端視窗寬度
func getTerminalWidth() int {
	width := 80 // 默認寬度

	// 嘗試獲取終端視窗大小
	fd := int(os.Stdout.Fd())
	if w, _, err := term.GetSize(fd); err == nil && w > 0 {
		width = w
	}

	return width
}

// 根據終端寬度決定顯示的列數
func determineColumnsToShow(colIndices []string, colWidths map[string]int, rowNameWidth, terminalWidth int) int {
	availableWidth := terminalWidth - rowNameWidth - 2 // 減去 RowNames 列和邊距

	if availableWidth <= 0 {
		return 0
	}

	// 計算每列需要的寬度（包括間距）
	var columnsToShow int
	usedWidth := 0

	for i, colIndex := range colIndices {
		colWidth := colWidths[colIndex] + 2 // 加上間距
		if usedWidth+colWidth <= availableWidth {
			usedWidth += colWidth
			columnsToShow = i + 1
		} else {
			break
		}
	}

	return columnsToShow
}

func (dt *DataTable) ShowTypes() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 構建資料地圖，但不使用 Data() 方法以避免死鎖
	dataMap := make(map[string][]any)
	for i, col := range dt.columns {
		key := generateColIndex(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// 取得所有的列索引並排序
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)

	// 獲取終端視窗寬度
	width := getTerminalWidth()
	// 生成資料表標題
	tableTitle := "DataTable 類型資訊"
	if len(dt.columns) > 0 && dt.columns[0].name != "" {
		tableTitle += ": " + dt.columns[0].name
	}

	// 在同一個鎖內獲取行數和列數，避免再次調用 Size() 導致死鎖
	rowCount := dt.getMaxColLength()
	colCount := len(dt.columns)
	tableSummary := fmt.Sprintf("(%d 行 x %d 列)", rowCount, colCount)

	// 資料表基本信息的顯示
	fmt.Printf("\033[1;36m%s\033[0m %s\n", tableTitle, tableSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// 如果 DataTable 為空，顯示一個消息
	if rowCount == 0 || colCount == 0 {
		fmt.Println("\033[3;33m(空表)\033[0m")
		return
	}
	// 計算每一列的最大寬度
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := fmt.Sprintf("%T", value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
		// 限制列寬度不超過特定值
		if colWidths[colIndex] > 25 {
			colWidths[colIndex] = 25
		}
	}

	// 計算 RowNames 的最大寬度，並顯示 RowIndex
	rowNames := make([]string, dt.getMaxColLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // 如果沒有名字則顯示為空
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // 加上 RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// 限制行名寬度不超過特定值
	if maxRowNameWidth > 25 {
		maxRowNameWidth = 25
	}

	// 根據當前視窗寬度動態調整要顯示的列數
	totalColsToShow := determineColumnsToShow(colIndices, colWidths, maxRowNameWidth, width)

	// 如果列數超過可顯示範圍，分頁顯示
	pageSize := totalColsToShow
	if pageSize <= 0 {
		pageSize = 1 // 至少顯示一列
	}

	// 計算需要多少頁
	totalPages := (len(colIndices) + pageSize - 1) / pageSize

	for page := 0; page < totalPages; page++ {
		start := page * pageSize
		end := (page + 1) * pageSize
		if end > len(colIndices) {
			end = len(colIndices)
		}

		currentPageCols := colIndices[start:end]

		if page > 0 {
			fmt.Println("\n\033[1;35m--- 繼續顯示類型 ---\033[0m")
		}

		if totalPages > 1 {
			pageInfo := fmt.Sprintf("--- 類型頁數 %d/%d ---", page+1, totalPages)
			fmt.Printf("\033[1;33m%s\033[0m\n", pageInfo)

			// 顯示頁面導航提示
			if page > 0 && page < totalPages-1 {
				fmt.Println("(查看更多請滾動屏幕)")
			}
		}

		// 打印列名
		fmt.Printf("\033[1;32m%-*s\033[0m", maxRowNameWidth+2, "RowNames")
		for _, colIndex := range currentPageCols {
			fmt.Printf(" \033[1;32m%-*s\033[0m", colWidths[colIndex]+1, TruncateString(colIndex, colWidths[colIndex]))
		}
		fmt.Println()

		// 打印分隔線
		printSeparator(maxRowNameWidth+2, currentPageCols, colWidths)

		// 打印行資料
		rowCount := dt.getMaxColLength()
		// 如果行數太多，只顯示前 20 行，中間省略，再顯示後 5 行
		if rowCount > 25 {
			// 顯示前 20 行
			printTypeRows(dataMap, 0, 20, rowNames, maxRowNameWidth, currentPageCols, colWidths) // 顯示省略號
			fmt.Printf("\033[1;33m%-*s\033[0m", maxRowNameWidth+2, "...")
			for range currentPageCols {
				fmt.Printf(" \033[1;33m%-*s\033[0m", colWidths[currentPageCols[0]]+1, "...")
			}
			fmt.Println()

			// 顯示後 5 行
			printTypeRows(dataMap, rowCount-5, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// 顯示資料摘要
			fmt.Printf("\n\033[3;36m共 %d 行資料，顯示了前 20 行和後 5 行\033[0m\n", rowCount)
		} else {
			// 行數不多，全部顯示
			printTypeRows(dataMap, 0, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)
		}

		// 如果是多頁，顯示页尾分隔线
		if totalPages > 1 {
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}
}

// 打印指定範圍的行（類型信息）
func printTypeRows(dataMap map[string][]any, start, end int, rowNames []string, maxRowNameWidth int, colIndices []string, colWidths map[string]int) {
	for rowIndex := start; rowIndex < end; rowIndex++ {
		rowName := ""
		if rowIndex < len(rowNames) {
			rowName = rowNames[rowIndex]
		}
		// 使用淺灰色顯示行名
		fmt.Printf("\033[1;37m%-*s\033[0m", maxRowNameWidth+2, TruncateString(rowName, maxRowNameWidth))

		for _, colIndex := range colIndices {
			value := "nil"
			var typeName string

			if rowIndex < len(dataMap[colIndex]) && dataMap[colIndex][rowIndex] != nil {
				rawValue := dataMap[colIndex][rowIndex]
				// 獲取更豐富的類型信息
				typeName = reflect.TypeOf(rawValue).String()
				value = typeName

				// 對於特殊類型添加額外信息
				switch v := rawValue.(type) {
				case []any:
					value = fmt.Sprintf("[]any(len=%d)", len(v))
				case []string:
					value = fmt.Sprintf("[]string(len=%d)", len(v))
				case map[string]any:
					value = fmt.Sprintf("map[string]any(len=%d)", len(v))
				case time.Time:
					value = "time.Time"
				}
			}

			// 根據類型使用不同的顏色
			typeColor := "\033[0m" // 默認顏色

			// 使用不同顏色區分不同類型
			if value == "nil" {
				typeColor = "\033[2;37m" // 淺灰色
			} else if strings.Contains(value, "int") || strings.Contains(value, "float") {
				typeColor = "\033[0;34m" // 藍色表示數字類型
			} else if strings.Contains(value, "string") {
				typeColor = "\033[0;32m" // 綠色表示字符串
			} else if strings.Contains(value, "bool") {
				typeColor = "\033[0;35m" // 紫色表示布爾值
			} else if strings.Contains(value, "map") || strings.Contains(value, "struct") {
				typeColor = "\033[0;33m" // 黃色表示複雜類型
			} else if strings.Contains(value, "slice") || strings.Contains(value, "array") || strings.Contains(value, "[]") {
				typeColor = "\033[0;36m" // 青色表示陣列或切片
			}

			fmt.Printf(" %s%-*s\033[0m", typeColor, colWidths[colIndex]+1, TruncateString(value, colWidths[colIndex]))
		}
		fmt.Println()
	}
}

// ========================= DataList ========================

// Show displays the content of DataList in a clean linear format.
// It adapts to terminal width and always displays in a linear format, not as a table.
// This method includes:
// - Basic information about the DataList (name, number of items)
// - Statistical summary (mean, min, max, range, etc.) if numeric data is present
// - Color-coded display of different data types (blue for numbers, green for strings, etc.)
// - Truncation for very large lists (showing only first 20 and last 5 items)
func (dl *DataList) Show() {
	// Safety check to prevent nil pointer
	if dl == nil {
		fmt.Println("\033[1;31mERROR: Unable to show a nil DataList\033[0m")
		return
	}

	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Get terminal window width
	width := getDataListTerminalWidth()

	// Generate data title
	dataTitle := "DataList"
	if dl.name != "" {
		dataTitle += ": " + dl.name
	}
	dataSummary := fmt.Sprintf("(%d items)", len(dl.data))
	// Display basic data information - 使用橘色作為 DataList 的主要顏色
	fmt.Printf("\033[1;33m%s\033[0m %s\n", dataTitle, dataSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Check if DataList is empty
	if len(dl.data) == 0 {
		fmt.Println("\033[3;33m(empty)\033[0m")
		return
	}
	// Display basic statistics
	showStatistics := true
	if showStatistics {
		// Try to calculate statistics
		mean, dlmin, max := dl.Mean(), dl.Min(), dl.Max()
		if !math.IsNaN(mean) && !math.IsNaN(dlmin) && !math.IsNaN(max) {
			// 使用橘色作為統計資訊的顏色
			fmt.Printf("\033[3;33m stat: mean=%.4g, min=%.4g, max=%.4g, range=%.4g\033[0m\n",
				mean, dlmin, max, max-dlmin)
			if len(dl.data) > 10 {
				fmt.Printf("\033[3;33m      SD=%.4g, median=%.4g\033[0m\n",
					dl.Stdev(), dl.Median())
			}
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}
	// Always show in linear format regardless of terminal width
	fmt.Println("\033[1;33mIndex  Value\033[0m")
	fmt.Println(strings.Repeat("-", min(width, 80)))

	// Calculate how many items to display
	totalItems := len(dl.data)
	maxDisplay := 25
	showAll := totalItems <= maxDisplay

	// Display items
	displayCount := totalItems
	if !showAll {
		displayCount = 20 // Show first 20 items
	}

	// Display items with appropriate formatting and colors
	for i := 0; i < displayCount; i++ {
		if i < totalItems {
			value := dl.data[i]
			strValue := FormatValue(value)

			// Color based on value type
			valueColor := "\033[0m" // Default color
			if value == nil {
				valueColor = "\033[2;37m" // Gray for nil
			} else {
				switch value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColor = "\033[0;34m" // Blue for numbers
				case string:
					valueColor = "\033[0;32m" // Green for strings
				case bool:
					valueColor = "\033[0;35m" // Purple for booleans
				}
			}

			fmt.Printf("\033[1;37m%-6d\033[0m %s%s\033[0m\n", i, valueColor, strValue)
		}
	}

	// If there are too many items, show ellipsis and the last few items
	if !showAll {
		fmt.Println("\033[1;33m...    ...\033[0m")

		// Show last 5 items
		for i := totalItems - 5; i < totalItems; i++ {
			value := dl.data[i]
			strValue := FormatValue(value)

			// Color based on value type
			valueColor := "\033[0m" // Default color
			if value == nil {
				valueColor = "\033[2;37m" // Gray for nil
			} else {
				switch value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColor = "\033[0;34m" // Blue for numbers
				case string:
					valueColor = "\033[0;32m" // Green for strings
				case bool:
					valueColor = "\033[0;35m" // Purple for booleans
				}
			}

			fmt.Printf("\033[1;37m%-6d\033[0m %s%s\033[0m\n", i, valueColor, strValue)
		}
		// Show data summary
		fmt.Printf("\n\033[3;33mTotal %d items, showing first 20 and last 5\033[0m\n", totalItems)
	}
}

// getDataListTerminalWidth gets the terminal window width, specifically for DataList
func getDataListTerminalWidth() int {
	width := 80 // Default width

	// Try to get terminal window size
	fd := int(os.Stdout.Fd())
	if w, _, err := term.GetSize(fd); err == nil && w > 0 {
		width = w
	}

	return width
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
