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
	// Build data map without using Data() method to avoid deadlock
	dataMap := make(map[string][]any)
	for i, col := range dt.columns {
		key := generateColIndex(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// Get all column indices and sort them
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)

	// Get terminal window width
	width := getTerminalWidth()
	// Generate table title
	tableTitle := "DataTable"
	if dt.name != "" {
		tableTitle += ": " + dt.name
	}
	// Get row and column counts inside the same lock to avoid deadlock with Size()
	rowCount := dt.getMaxColLength()
	colCount := len(dt.columns)
	tableSummary := fmt.Sprintf("(%d rows x %d columns)", rowCount, colCount)
	// Display table basic info - using DataTable primary color
	fmt.Printf("\033[1;36m%s\033[0m \033[3;36m%s\033[0m\n", tableTitle, tableSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Handle empty table
	if rowCount == 0 || colCount == 0 {
		fmt.Println("\033[2;37m(empty)\033[0m")
		return
	}
	// Calculate the maximum width for each column
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := FormatValue(value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
		// Limit column width to a specific value
		if colWidths[colIndex] > 30 {
			colWidths[colIndex] = 30
		}
	}

	// Calculate the maximum width of RowNames and display RowIndex
	rowNames := make([]string, dt.getMaxColLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // Display empty if no name
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // Add RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// Limit row name width to a specific value
	if maxRowNameWidth > 25 {
		maxRowNameWidth = 25
	}
	// Try to display some basic statistics
	if rowCount > 0 && colCount > 0 {
		// Display basic statistics for each column
		hasNumbers := false
		statsInfo := "\033[3;36m stat: "

		for _, colIndex := range colIndices[:min(3, len(colIndices))] { // Only show statistics for the first three columns
			// Try to convert columns to numbers for statistical calculations
			dl := NewDataList(dataMap[colIndex])
			dl.ParseNumbers() // Try to parse as numbers

			mean, colMin, colMax := dl.Mean(), dl.Min(), dl.Max()
			if !math.IsNaN(mean) && !math.IsNaN(colMin) && !math.IsNaN(colMax) {
				hasNumbers = true
				shortColName := strings.Split(colIndex, "(")[0] // Only use short column name
				statsInfo += fmt.Sprintf("%s(mean=%.4g, range=[%.4g, %.4g]) ",
					shortColName, mean, colMin, colMax)
			}
		}

		if hasNumbers {
			// If there are numeric columns, display statistics
			statsInfo += "\033[0m"
			fmt.Println(statsInfo)
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}

	// Dynamically adjust the number of columns to display based on current window width
	totalColsToShow := determineColumnsToShow(colIndices, colWidths, maxRowNameWidth, width)

	// If columns exceed display range, paginate display
	pageSize := totalColsToShow
	if pageSize <= 0 {
		pageSize = 1 // Display at least one column
	}

	// Calculate how many pages are needed
	totalPages := (len(colIndices) + pageSize - 1) / pageSize

	for page := 0; page < totalPages; page++ {
		start := page * pageSize
		end := (page + 1) * pageSize
		if end > len(colIndices) {
			end = len(colIndices)
		}

		currentPageCols := colIndices[start:end]

		if page > 0 {
			fmt.Println("\n\033[1;35m--- Continue Display ---\033[0m")
		}
		if totalPages > 1 {
			pageInfo := fmt.Sprintf("--- Page %d/%d ---", page+1, totalPages)
			fmt.Printf("\033[1;36m%s\033[0m\n", pageInfo)

			// Display page navigation prompt
			if page > 0 && page < totalPages-1 {
				fmt.Println("(Scroll screen to see more)")
			}
		}
		// Print column names - using header text color
		fmt.Printf("\033[1;32m%-*s\033[0m", maxRowNameWidth+2, "RowNames")
		for _, colIndex := range currentPageCols {
			fmt.Printf(" \033[1;32m%-*s\033[0m", colWidths[colIndex]+1, TruncateString(colIndex, colWidths[colIndex]))
		}
		fmt.Println()

		// Print separator
		printSeparator(maxRowNameWidth+2, currentPageCols, colWidths)

		// Print row data
		rowCount := dt.getMaxColLength()
		// If there are too many rows, only show the first 20 rows, ellipsis in the middle, then the last 5 rows
		if rowCount > 25 {
			// Show first 20 rows
			printRowsColored(dataMap, 0, 20, rowNames, maxRowNameWidth, currentPageCols, colWidths) // Show ellipsis
			fmt.Printf("\033[1;36m%-*s\033[0m", maxRowNameWidth+2, "...")
			for range currentPageCols {
				fmt.Printf(" \033[1;36m%-*s\033[0m", colWidths[currentPageCols[0]]+1, "...")
			}
			fmt.Println()

			// Show last 5 rows
			printRowsColored(dataMap, rowCount-5, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)
			// Show data summary - using secondary color
			fmt.Printf("\n\033[3;36mTotal %d rows of data, showing first 20 and last 5\033[0m\n", rowCount)
		} else {
			// Not many rows, show all
			printRowsColored(dataMap, 0, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)
		}

		// If multiple pages, show footer separator
		if totalPages > 1 {
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}
}

// Print specified range of rows (with color)
func printRowsColored(dataMap map[string][]any, start, end int, rowNames []string, maxRowNameWidth int, colIndices []string, colWidths map[string]int) {
	for rowIndex := start; rowIndex < end; rowIndex++ {
		rowName := ""
		if rowIndex < len(rowNames) {
			rowName = rowNames[rowIndex]
		}
		// Use row name color
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

			// Use different colors based on value type
			valueColor := "\033[0m" // Default color

			// If value is nil, use gray
			if rawValue == nil {
				valueColor = "\033[2;37m" // Nil value color
			} else {
				switch rawValue.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColor = "\033[0;34m" // Numeric data color
				case string:
					valueColor = "\033[0;32m" // Text data color
				case bool:
					valueColor = "\033[0;35m" // Purple for boolean values
				}
			}

			fmt.Printf(" %s%-*s\033[0m", valueColor, colWidths[colIndex]+1, TruncateString(value, colWidths[colIndex]))
		}
		fmt.Println()
	}
}

// Print separator
func printSeparator(rowNameWidth int, colIndices []string, colWidths map[string]int) {
	fmt.Print(strings.Repeat("-", rowNameWidth))
	for _, colIndex := range colIndices {
		fmt.Print(" " + strings.Repeat("-", colWidths[colIndex]+1))
	}
	fmt.Println()
}

// Get terminal window width
func getTerminalWidth() int {
	width := 80 // Default width

	// Try to get terminal window size
	fd := int(os.Stdout.Fd())
	if w, _, err := term.GetSize(fd); err == nil && w > 0 {
		width = w
	}

	return width
}

// Determine the number of columns to display based on terminal width
func determineColumnsToShow(colIndices []string, colWidths map[string]int, rowNameWidth, terminalWidth int) int {
	availableWidth := terminalWidth - rowNameWidth - 2 // Subtract RowNames column and margins

	if availableWidth <= 0 {
		return 0
	}

	// Calculate the width required for each column (including spacing)
	var columnsToShow int
	usedWidth := 0

	for i, colIndex := range colIndices {
		colWidth := colWidths[colIndex] + 2 // Add spacing
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

	// Build data map without using Data() method to avoid deadlock
	dataMap := make(map[string][]any)
	for i, col := range dt.columns {
		key := generateColIndex(i)
		if col.name != "" {
			key += fmt.Sprintf("(%s)", col.name)
		}
		dataMap[key] = col.data
	}

	// Get all column indices and sort them
	var colIndices []string
	for colIndex := range dataMap {
		colIndices = append(colIndices, colIndex)
	}
	sort.Strings(colIndices)
	// Get terminal window width
	width := getTerminalWidth()
	// Generate table title
	tableTitle := "DataTable Type Info"
	if len(dt.columns) > 0 && dt.columns[0].name != "" {
		tableTitle += ": " + dt.columns[0].name
	}

	// Get row and column counts inside the same lock to avoid deadlock with Size()
	rowCount := dt.getMaxColLength()
	colCount := len(dt.columns)
	tableSummary := fmt.Sprintf("(%d rows x %d columns)", rowCount, colCount)
	// Display table basic info
	fmt.Printf("\033[1;36m%s\033[0m \033[3;36m%s\033[0m\n", tableTitle, tableSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))
	// Handle empty table
	if rowCount == 0 || colCount == 0 {
		fmt.Println("\033[3;36m(empty)\033[0m")
		return
	}
	// Calculate the maximum width for each column
	colWidths := make(map[string]int)
	for _, colIndex := range colIndices {
		colWidths[colIndex] = len(colIndex)
		for _, value := range dataMap[colIndex] {
			valueStr := fmt.Sprintf("%T", value)
			if len(valueStr) > colWidths[colIndex] {
				colWidths[colIndex] = len(valueStr)
			}
		}
		// Limit column width to a specific value
		if colWidths[colIndex] > 25 {
			colWidths[colIndex] = 25
		}
	}

	// Calculate the maximum width of RowNames and display RowIndex
	rowNames := make([]string, dt.getMaxColLength())
	maxRowNameWidth := len("RowNames")
	for i := range rowNames {
		if rowName, exists := dt.getRowNameByIndex(i); exists {
			rowNames[i] = rowName
		} else {
			rowNames[i] = "" // Display empty if no name
		}
		rowNames[i] = fmt.Sprintf("%d: %s", i, rowNames[i]) // Add RowIndex
		if len(rowNames[i]) > maxRowNameWidth {
			maxRowNameWidth = len(rowNames[i])
		}
	}

	// Limit row name width to a specific value
	if maxRowNameWidth > 25 {
		maxRowNameWidth = 25
	}

	// Dynamically adjust the number of columns to display based on current window width
	totalColsToShow := determineColumnsToShow(colIndices, colWidths, maxRowNameWidth, width)

	// If columns exceed display range, paginate display
	pageSize := totalColsToShow
	if pageSize <= 0 {
		pageSize = 1 // Display at least one column
	}

	// Calculate how many pages are needed
	totalPages := (len(colIndices) + pageSize - 1) / pageSize

	for page := 0; page < totalPages; page++ {
		start := page * pageSize
		end := (page + 1) * pageSize
		if end > len(colIndices) {
			end = len(colIndices)
		}

		currentPageCols := colIndices[start:end]

		if page > 0 {
			fmt.Println("\n\033[1;35m--- Continue Type Display ---\033[0m")
		}
		if totalPages > 1 {
			pageInfo := fmt.Sprintf("--- Type Page %d/%d ---", page+1, totalPages)
			fmt.Printf("\033[1;36m%s\033[0m\n", pageInfo)

			// Display page navigation prompt
			if page > 0 && page < totalPages-1 {
				fmt.Println("(Scroll screen to see more)")
			}
		}

		// Print column names
		fmt.Printf("\033[1;32m%-*s\033[0m", maxRowNameWidth+2, "RowNames")
		for _, colIndex := range currentPageCols {
			fmt.Printf(" \033[1;32m%-*s\033[0m", colWidths[colIndex]+1, TruncateString(colIndex, colWidths[colIndex]))
		}
		fmt.Println()

		// Print separator
		printSeparator(maxRowNameWidth+2, currentPageCols, colWidths)

		// Print row data
		rowCount := dt.getMaxColLength()
		// If there are too many rows, only show the first 20 rows, ellipsis in the middle, then the last 5 rows
		if rowCount > 25 {
			// Show first 20 rows
			printTypeRows(dataMap, 0, 20, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// Show ellipsis
			fmt.Printf("\033[1;36m%-*s\033[0m", maxRowNameWidth+2, "...")
			for range currentPageCols {
				fmt.Printf(" \033[1;36m%-*s\033[0m", colWidths[currentPageCols[0]]+1, "...")
			}
			fmt.Println()

			// Show last 5 rows
			printTypeRows(dataMap, rowCount-5, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// Show data summary
			fmt.Printf("\n\033[3;36mTotal %d rows of data, showing first 20 and last 5\033[0m\n", rowCount)
		} else {
			// Not many rows, show all
			printTypeRows(dataMap, 0, rowCount, rowNames, maxRowNameWidth, currentPageCols, colWidths)
		}

		// If multiple pages, show footer separator
		if totalPages > 1 {
			fmt.Println(strings.Repeat("-", min(width, 80)))
		}
	}
}

// Print specified range of rows (type information)
func printTypeRows(dataMap map[string][]any, start, end int, rowNames []string, maxRowNameWidth int, colIndices []string, colWidths map[string]int) {
	for rowIndex := start; rowIndex < end; rowIndex++ {
		rowName := ""
		if rowIndex < len(rowNames) {
			rowName = rowNames[rowIndex]
		}
		// Use light gray color for row names
		fmt.Printf("\033[1;37m%-*s\033[0m", maxRowNameWidth+2, TruncateString(rowName, maxRowNameWidth))

		for _, colIndex := range colIndices {
			value := "nil"
			var typeName string

			if rowIndex < len(dataMap[colIndex]) && dataMap[colIndex][rowIndex] != nil {
				rawValue := dataMap[colIndex][rowIndex]
				// Get richer type information
				typeName = reflect.TypeOf(rawValue).String()
				value = typeName

				// Add extra information for special types
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

			// Use different colors based on type
			typeColor := "\033[0m" // Default color

			// Use different colors to distinguish types
			if value == "nil" {
				typeColor = "\033[2;37m" // Light gray
			} else if strings.Contains(value, "int") || strings.Contains(value, "float") {
				typeColor = "\033[0;34m" // Blue for numeric types
			} else if strings.Contains(value, "string") {
				typeColor = "\033[0;32m" // Green for strings
			} else if strings.Contains(value, "bool") {
				typeColor = "\033[0;35m" // Purple for booleans
			} else if strings.Contains(value, "map") || strings.Contains(value, "struct") {
				typeColor = "\033[0;33m" // Yellow for complex types
			} else if strings.Contains(value, "slice") || strings.Contains(value, "array") || strings.Contains(value, "[]") {
				typeColor = "\033[0;36m" // Cyan for arrays or slices
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
	// Display basic data information - using DataList primary color
	fmt.Printf("\033[1;33m%s\033[0m \033[3;33m%s\033[0m\n", dataTitle, dataSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Check if DataList is empty
	if len(dl.data) == 0 {
		fmt.Println("\033[2;37m(empty)\033[0m")
		return
	}
	// Display basic statistics
	showStatistics := true
	if showStatistics {
		// Try to calculate statistics
		mean, dlmin, max := dl.Mean(), dl.Min(), dl.Max()
		if !math.IsNaN(mean) && !math.IsNaN(dlmin) && !math.IsNaN(max) { // Using secondary color to display statistics
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
	fmt.Println("\033[1;32mIndex\033[0m  \033[1;32mValue\033[0m")
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
				valueColor = "\033[2;37m" // Nil value color
			} else {
				switch value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColor = "\033[0;34m" // Numeric data color
				case string:
					valueColor = "\033[0;32m" // Text data color
				case bool:
					valueColor = "\033[0;35m" // Purple for boolean values
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
				valueColor = "\033[2;37m" // Nil value color
			} else {
				switch value.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColor = "\033[0;34m" // Numeric data color
				case string:
					valueColor = "\033[0;32m" // Text data color
				case bool:
					valueColor = "\033[0;35m" // Purple for boolean values
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

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
