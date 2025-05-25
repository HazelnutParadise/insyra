package insyra

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// ======================== DataTable ========================

// Show displays the content of the DataTable in a formatted way.
// For more control over which rows to display, use ShowRange.
func (dt *DataTable) Show() {
	// Call ShowRange without any parameters to show all rows
	dt.ShowRange()
}

// ShowRange displays the DataTable with a specified range of rows.
// startEnd is an optional parameter that can be [start, end] to specify the range of rows to display.
// if startEnd is not provided, all rows will be displayed.
// if only one value is provided, there are two behaviors:
// - if positive, it shows the first N rows (e.g., ShowRange(5) shows first 5 rows)
// - if negative, it shows the last N rows (e.g., ShowRange(-5) shows last 5 rows)
// For two parameters [start, end], it shows rows from index start (inclusive) to index end (exclusive).
// If end is nil, it shows rows from index start to the end of the table.
// Example: dt.ShowRange() - shows all rows
// Example: dt.ShowRange(5) - shows the first 5 rows
// Example: dt.ShowRange(-5) - shows the last 5 rows
// Example: dt.ShowRange(2, 10) - shows rows with indices 2 to 9 (not including 10)
// Example: dt.ShowRange(2, nil) - shows rows from index 2 to the end of the table
func (dt *DataTable) ShowRange(startEnd ...interface{}) {
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
	// Adjust start and end indices based on input parameters
	start, end := 0, rowCount
	if len(startEnd) > 0 {
		if len(startEnd) == 1 {
			if countVal, ok := startEnd[0].(int); ok {
				if countVal < 0 {
					start = max(0, rowCount+countVal)
					end = rowCount
				} else {
					start = 0
					end = min(countVal, rowCount)
				}
			}
		} else if len(startEnd) >= 2 {
			if startVal, ok := startEnd[0].(int); ok {
				start = startVal
				if start < 0 {
					start = rowCount + start
				}
			}

			if startEnd[1] == nil {
				end = rowCount
			} else if endVal, ok := startEnd[1].(int); ok {
				end = endVal
				if end < 0 {
					end = rowCount + end
				}
			}
		}

		if end > rowCount {
			end = rowCount
		}

		if start < 0 {
			start = 0
		}

		if start >= end {
			// Nothing to display if start is greater than or equal to end
			fmt.Println("\033[2;37m(empty range)\033[0m")
			return
		}
	}

	// Display range information in the summary if it's a subset
	rangeInfo := ""
	if start > 0 || end < rowCount {
		rangeInfo = fmt.Sprintf(" [showing rows %d to %d]", start, end-1)
	}

	tableSummary := fmt.Sprintf("(%d rows x %d columns)%s", rowCount, colCount, rangeInfo)
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

	// Try to display some basic statistics for the visible range
	if end-start > 0 && colCount > 0 {
		// Display basic statistics for each column
		hasNumbers := false
		statsInfo := "\033[3;36m stat"
		if start > 0 || end < rowCount {
			statsInfo += " (selected range)"
		}
		statsInfo += ": "

		for _, colIndex := range colIndices[:min(3, len(colIndices))] { // Only show statistics for the first three columns
			// Create a subset of data for the visible range
			rangeData := make([]any, 0, end-start)
			for i := start; i < end && i < len(dataMap[colIndex]); i++ {
				rangeData = append(rangeData, dataMap[colIndex][i])
			}

			// Check if the column contains numeric data before attempting statistics
			hasNumericData := false
			for _, val := range rangeData {
				if val != nil {
					switch v := val.(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
						hasNumericData = true
					case string:
						// Check if string can be parsed as number using strconv directly
						if _, err := strconv.ParseFloat(v, 64); err == nil {
							hasNumericData = true
							break
						}
					}
					if hasNumericData {
						break
					}
				}
			}

			// Only attempt statistical calculations if numeric data is found
			if hasNumericData {
				// Create a temporary DataList with only numeric values for statistics
				numericValues := make([]any, 0)
				for _, val := range rangeData {
					if val != nil {
						switch v := val.(type) {
						case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
							numericValues = append(numericValues, v)
						case string:
							if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
								numericValues = append(numericValues, floatVal)
							}
						}
					}
				}

				if len(numericValues) > 0 {
					dl := NewDataList(numericValues...)
					mean, colMin, colMax := dl.Mean(), dl.Min(), dl.Max()
					if !math.IsNaN(mean) && !math.IsNaN(colMin) && !math.IsNaN(colMax) {
						hasNumbers = true
						shortColName := strings.Split(colIndex, "(")[0] // Only use short column name
						statsInfo += fmt.Sprintf("%s(mean=%.4g, range=[%.4g, %.4g]) ",
							shortColName, mean, colMin, colMax)
					}
				}
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
		pageStart := page * pageSize
		pageEnd := (page + 1) * pageSize
		if pageEnd > len(colIndices) {
			pageEnd = len(colIndices)
		}

		currentPageCols := colIndices[pageStart:pageEnd]

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
		// Print row data for the specified range
		selectedRowCount := end - start

		// Check if range was explicitly specified
		explicitRangeSpecified := len(startEnd) > 0

		// If there are too many rows in the selected range, only show first 20 and last 5
		// UNLESS a range was explicitly specified by the user
		if selectedRowCount > 25 && !explicitRangeSpecified {
			// Show first 20 rows
			printRowsColored(dataMap, start, start+20, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// Show ellipsis
			fmt.Printf("\033[1;36m%-*s\033[0m", maxRowNameWidth+2, "...")
			for range currentPageCols {
				fmt.Printf(" \033[1;36m%-*s\033[0m", colWidths[currentPageCols[0]]+1, "...")
			}
			fmt.Println()

			// Show last 5 rows
			printRowsColored(dataMap, end-5, end, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// Show data summary - using secondary color
			fmt.Printf("\n\033[3;36mDisplaying %d rows (from row %d to row %d)\033[0m\n",
				selectedRowCount, start, end-1)
		} else {
			// Either not many rows or user explicitly requested the range,
			// so show all rows in the range without truncation
			printRowsColored(dataMap, start, end, rowNames, maxRowNameWidth, currentPageCols, colWidths)
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

// ShowTypes displays the data types of each element in the DataTable.
// For more control over which rows to display, use ShowTypesRange.
func (dt *DataTable) ShowTypes() {
	// Call ShowTypesRange without any parameters to show all rows
	dt.ShowTypesRange()
}

// ShowTypesRange displays the data types of each element in the DataTable within a specified range of rows.
// startEnd is an optional parameter that can be [start, end] to specify the range of rows to display.
// if startEnd is not provided, all rows will be displayed.
// if only one value is provided, there are two behaviors:
// - if positive, it shows the first N rows (e.g., ShowTypesRange(5) shows first 5 rows)
// - if negative, it shows the last N rows (e.g., ShowTypesRange(-5) shows last 5 rows)
// For two parameters [start, end], it shows rows from index start (inclusive) to index end (exclusive).
// If end is nil, it shows rows from index start to the end of the table.
// Example: dt.ShowTypesRange() - shows all rows
// Example: dt.ShowTypesRange(5) - shows the first 5 rows
// Example: dt.ShowTypesRange(-5) - shows the last 5 rows
// Example: dt.ShowTypesRange(2, 10) - shows rows with indices 2 to 9 (not including 10)
// Example: dt.ShowTypesRange(2, nil) - shows rows from index 2 to the end of the table
func (dt *DataTable) ShowTypesRange(startEnd ...interface{}) {
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
	// Adjust start and end indices based on input parameters
	start, end := 0, rowCount
	if len(startEnd) > 0 {
		if len(startEnd) == 1 {
			if countVal, ok := startEnd[0].(int); ok {
				if countVal < 0 {
					start = max(0, rowCount+countVal)
					end = rowCount
				} else {
					start = 0
					end = min(countVal, rowCount)
				}
			}
		} else if len(startEnd) >= 2 {
			if startVal, ok := startEnd[0].(int); ok {
				start = startVal
				if start < 0 {
					start = rowCount + start
				}
			}

			if startEnd[1] == nil {
				end = rowCount
			} else if endVal, ok := startEnd[1].(int); ok {
				end = endVal
				if end < 0 {
					end = rowCount + end
				}
			}
		}

		if end > rowCount {
			end = rowCount
		}

		if start < 0 {
			start = 0
		}
		if start >= end {
			// Nothing to display if start is greater than or equal to end
			fmt.Println("\033[2;37m(empty range)\033[0m")
			return
		}
	}

	// Display range information in the summary if it's a subset
	rangeInfo := ""
	if start > 0 || end < rowCount {
		rangeInfo = fmt.Sprintf(" [showing rows %d to %d]", start, end-1)
	}

	tableSummary := fmt.Sprintf("(%d rows x %d columns)%s", rowCount, colCount, rangeInfo)
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
		pageStart := page * pageSize
		pageEnd := (page + 1) * pageSize
		if pageEnd > len(colIndices) {
			pageEnd = len(colIndices)
		}

		currentPageCols := colIndices[pageStart:pageEnd]

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
		// Print row data for the specified range
		selectedRowCount := end - start

		// Check if range was explicitly specified
		explicitRangeSpecified := len(startEnd) > 0

		// If there are too many rows in the selected range, only show first 20 and last 5
		// UNLESS a range was explicitly specified by the user
		if selectedRowCount > 25 && !explicitRangeSpecified {
			// Show first 20 rows
			printTypeRows(dataMap, start, start+20, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// Show ellipsis
			fmt.Printf("\033[1;36m%-*s\033[0m", maxRowNameWidth+2, "...")
			for range currentPageCols {
				fmt.Printf(" \033[1;36m%-*s\033[0m", colWidths[currentPageCols[0]]+1, "...")
			}
			fmt.Println()

			// Show last 5 rows
			printTypeRows(dataMap, end-5, end, rowNames, maxRowNameWidth, currentPageCols, colWidths)

			// Show data summary
			fmt.Printf("\n\033[3;36mDisplaying %d rows (from row %d to row %d)\033[0m\n",
				selectedRowCount, start, end-1)
		} else {
			// Either not many rows or user explicitly requested the range,
			// so show all rows in the range without truncation
			printTypeRows(dataMap, start, end, rowNames, maxRowNameWidth, currentPageCols, colWidths)
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
// For more control over which items to display, use ShowRange.
func (dl *DataList) Show() {
	// Call ShowRange without any parameters to show all items
	dl.ShowRange()
}

// ShowRange displays the content of DataList within a specified range in a clean linear format.
// It adapts to terminal width and always displays in a linear format, not as a table.
// startEnd is an optional parameter that can be [start, end] to specify the range of items to display.
// if startEnd is not provided, all items will be displayed.
// if only one value is provided, there are two behaviors:
// - if positive, it shows the first N items (e.g., ShowRange(5) shows first 5 items)
// - if negative, it shows the last N items (e.g., ShowRange(-5) shows last 5 items)
// For two parameters [start, end], it shows items from index start (inclusive) to index end (exclusive).
// If end is nil, it shows items from index start to the end of the list.
// Example: dl.ShowRange() - shows all items
// Example: dl.ShowRange(5) - shows the first 5 items
// Example: dl.ShowRange(-5) - shows the last 5 items
// Example: dl.ShowRange(2, 10) - shows items with indices 2 to 9 (not including 10)
// Example: dl.ShowRange(2, -1) - shows items from index 2 to the end of the list
// Example: dl.ShowRange(2, nil) - shows items from index 2 to the end of the list
func (dl *DataList) ShowRange(startEnd ...any) {
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

	// Get total items count
	totalItems := len(dl.data)

	// Adjust start and end indices based on input parameters
	start, end := 0, totalItems
	if len(startEnd) > 0 {
		if len(startEnd) == 1 {
			if countVal, ok := startEnd[0].(int); ok {
				if countVal < 0 {
					start = max(0, totalItems+countVal)
					end = totalItems
				} else {
					start = 0
					end = min(countVal, totalItems)
				}
			}
		} else if len(startEnd) >= 2 {
			if startVal, ok := startEnd[0].(int); ok {
				start = startVal
				if start < 0 {
					// 對於負索引，將其轉換為相對於總行數的索引
					start = totalItems + start
				}
			}

			if startEnd[1] == nil {
				// 如果第二個參數是 nil，表示顯示到最後一個元素
				end = totalItems
			} else if endVal, ok := startEnd[1].(int); ok {
				end = endVal
				if end < 0 {
					// 對於負索引，將其轉換為相對於總行數的索引
					end = totalItems + end
				}
			}

			if end > totalItems {
				end = totalItems
			}
		}

		// 確保起始索引不會小於0（適用於所有情況）
		if start < 0 {
			start = 0
		}

		if start >= end {
			// Nothing to display if start is greater than or equal to end
			fmt.Printf("\033[1;33m%s\033[0m \033[3;33m(%d items)\033[0m\n", dataTitle, totalItems)
			fmt.Println(strings.Repeat("=", min(width, 80)))
			fmt.Println("\033[2;37m(empty range)\033[0m")
			return
		}
	}

	// Display range information in the summary if it's a subset
	rangeInfo := ""
	if start > 0 || end < totalItems {
		rangeInfo = fmt.Sprintf(" [showing items %d to %d]", start, end-1)
	}

	dataSummary := fmt.Sprintf("(%d items)%s", totalItems, rangeInfo)
	// Display basic data information - using DataList primary color
	fmt.Printf("\033[1;33m%s\033[0m \033[3;33m%s\033[0m\n", dataTitle, dataSummary)
	fmt.Println(strings.Repeat("=", min(width, 80)))

	// Check if DataList is empty
	if totalItems == 0 {
		fmt.Println("\033[2;37m(empty)\033[0m")
		return
	}

	// Display basic statistics for the selected range
	showStatistics := true
	if showStatistics && (end-start) > 0 {
		// Create a subset of data for the visible range
		rangeData := make([]any, 0, end-start)
		for i := start; i < end; i++ {
			rangeData = append(rangeData, dl.data[i])
		}

		// Check if the data contains numeric values before attempting statistics
		hasNumericData := false
		for _, val := range rangeData {
			if val != nil {
				switch v := val.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					hasNumericData = true
				case string:
					// Check if string can be parsed as number using strconv directly
					if _, err := strconv.ParseFloat(v, 64); err == nil {
						hasNumericData = true
						break
					}
				}
				if hasNumericData {
					break
				}
			}
		}

		// Only attempt statistical calculations if numeric data is found
		if hasNumericData {
			// Create a temporary DataList with only numeric values for statistics
			numericValues := make([]any, 0)
			for _, val := range rangeData {
				if val != nil {
					switch v := val.(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
						numericValues = append(numericValues, v)
					case string:
						if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
							numericValues = append(numericValues, floatVal)
						}
					}
				}
			}

			if len(numericValues) > 0 {
				rangeDl := NewDataList(numericValues...)
				// Try to calculate statistics for the range
				mean, dlmin, max := rangeDl.Mean(), rangeDl.Min(), rangeDl.Max()
				if !math.IsNaN(mean) && !math.IsNaN(dlmin) && !math.IsNaN(max) { // Using secondary color to display statistics
					statsLabel := "stat"
					if start > 0 || end < totalItems {
						statsLabel = "stat (selected range)"
					}
					fmt.Printf("\033[3;33m %s: mean=%.4g, min=%.4g, max=%.4g, range=%.4g\033[0m\n",
						statsLabel, mean, dlmin, max, max-dlmin)
					if len(numericValues) > 10 {
						fmt.Printf("\033[3;33m      SD=%.4g, median=%.4g\033[0m\n",
							rangeDl.Stdev(), rangeDl.Median())
					}
					fmt.Println(strings.Repeat("-", min(width, 80)))
				}
			}
		}
	}

	// Always show in linear format regardless of terminal width
	fmt.Println("\033[1;32mIndex\033[0m  \033[1;32mValue\033[0m")
	fmt.Println(strings.Repeat("-", min(width, 80)))
	// Calculate how many items to display in the range
	selectedItems := end - start
	maxDisplay := 25

	// Check if range was explicitly specified
	explicitRangeSpecified := len(startEnd) > 0

	// Show all items if either:
	// 1. The selected range is small (less than maxDisplay), or
	// 2. The user explicitly specified a range (even if it's large)
	showAll := selectedItems <= maxDisplay || explicitRangeSpecified

	// Display items
	displayCount := selectedItems
	if !showAll {
		displayCount = 20 // Show first 20 items
	}

	// Display items with appropriate formatting and colors
	for i := 0; i < displayCount; i++ {
		itemIndex := start + i
		if itemIndex < end {
			value := dl.data[itemIndex]
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

			fmt.Printf("\033[1;37m%-6d\033[0m %s%s\033[0m\n", itemIndex, valueColor, strValue)
		}
	}

	// If there are too many items in the range, show ellipsis and the last few items
	if !showAll {
		fmt.Println("\033[1;33m...    ...\033[0m")

		// Show last 5 items from the range
		for i := end - 5; i < end; i++ {
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
		fmt.Printf("\n\033[3;33mDisplaying %d items (from index %d to index %d)\033[0m\n",
			selectedItems, start, end-1)
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
