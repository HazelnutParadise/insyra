package insyra

import (
	"cmp"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/utils"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type showable interface {
	ShowRange(startEnd ...any)
}

// Show displays the content of any showable object with a label.
// Automatically deals with nil objects.
func Show(label string, object showable, startEnd ...any) {
	if object == nil {
		fmt.Printf("%s: %s\n\n", label, colorText("2;37", "(nil)"))
		return
	}
	fmt.Printf("%s\n", colorText("1;35", fmt.Sprintf("--- Showing: %s ---", label)))
	object.ShowRange(startEnd...)
}

func colorize(code string, text any) string {
	if code == "" {
		return fmt.Sprintf("%v", text)
	}
	return colorText(code, text)
}

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
func (dt *DataTable) ShowRange(startEnd ...any) {
	dt.AtomicDo(func(dt *DataTable) {

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

		utils.ParallelSortStableFunc(colIndices, func(a, b string) int {
			prefixA := a
			if idx := strings.Index(a, "("); idx != -1 {
				prefixA = a[:idx]
			}

			prefixB := b
			if idx := strings.Index(b, "("); idx != -1 {
				prefixB = b[:idx]
			}
			return cmp.Compare(ParseColIndex(prefixA), ParseColIndex(prefixB))
		})

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
				fmt.Println(colorText("2;37", "(empty range)"))
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
		fmt.Printf("%s %s\n", colorText("1;36", tableTitle), colorText("3;36", tableSummary))
		fmt.Println(strings.Repeat("=", min(width, 80)))

		// Handle empty table
		if rowCount == 0 || colCount == 0 {
			fmt.Println(colorText("2;37", "(empty)"))
			return
		}
		// Compute table layout (column widths, row names, and max row name width)
		colWidths, rowNames, maxRowNameWidth := prepareTableLayout(dt, dataMap, colIndices)

		// Try to display some basic statistics for the visible range
		if end-start > 0 && colCount > 0 {
			// Display basic statistics for each column
			hasNumbers := false
			statsLabel := "stat"
			if start > 0 || end < rowCount {
				statsLabel += " (selected range)"
			}
			var statParts []string

			for _, colIndex := range colIndices[:min(3, len(colIndices))] {
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
							statParts = append(statParts, fmt.Sprintf("%s(mean=%.2f, range=[%.2f, %.2f]) ",
								shortColName, mean, colMin, colMax))
						}
					}
				}
			}

			if hasNumbers {
				// If there are numeric columns, display statistics
				statsContent := strings.Join(statParts, "")
				fmt.Println(colorText("3;36", fmt.Sprintf(" %s: %s", statsLabel, statsContent)))
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
				fmt.Println("\n" + colorText("1;35", "--- Continue Display ---"))
			}
			if totalPages > 1 {
				pageInfo := fmt.Sprintf("--- Page %d/%d ---", page+1, totalPages)
				fmt.Println(colorText("1;36", pageInfo))

				// Display page navigation prompt
				if page > 0 && page < totalPages-1 {
					fmt.Println("(Scroll screen to see more)")
				}
			} // Print column names - using header text color
			// Print header with proper alignment using runewidth
			fmt.Print(colorText("1;32", runewidth.FillRight("RowNames", maxRowNameWidth+2)))
			for _, colIndex := range currentPageCols {
				label := utils.TruncateString(colIndex, colWidths[colIndex])
				fmt.Print(" " + colorText("1;32", runewidth.FillRight(label, colWidths[colIndex]+1)))
			}
			fmt.Println()

			// Print separator aligned to header widths
			fmt.Print(strings.Repeat("-", maxRowNameWidth+2))
			for _, colIndex := range currentPageCols {
				fmt.Print(" " + strings.Repeat("-", colWidths[colIndex]+1))
			}
			fmt.Println()
			// Print row data for the specified range
			selectedRowCount := end - start

			// Check if range was explicitly specified
			explicitRangeSpecified := len(startEnd) > 0
			// If there are too many rows in the selected range, only show first 20 and last 5
			// UNLESS a range was explicitly specified by the user
			if selectedRowCount > 25 && !explicitRangeSpecified {
				// Show first 20 rows
				printRowsColored(dataMap, start, start+20, rowNames, maxRowNameWidth, currentPageCols, colWidths)

				// Show ellipsis line aligned to columns
				fmt.Print(colorText("1;36", runewidth.FillRight("...", maxRowNameWidth+2)))
				for _, idx := range currentPageCols {
					fmt.Print(" " + runewidth.FillRight("...", colWidths[idx]+1))
				}
				fmt.Println()

				// Show last 5 rows
				printRowsColored(dataMap, end-5, end, rowNames, maxRowNameWidth, currentPageCols, colWidths)

				// Show data summary - using secondary color
				fmt.Printf("\n%s\n", colorText("3;36", fmt.Sprintf("Displaying %d rows (from row %d to row %d)",
					selectedRowCount, start, end-1)))
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
	})
	fmt.Println()
}

// Print specified range of rows (with color)
func printRowsColored(dataMap map[string][]any, start, end int, rowNames []string, maxRowNameWidth int, colIndices []string, colWidths map[string]int) {
	for rowIndex := start; rowIndex < end; rowIndex++ {
		rowName := ""
		if rowIndex < len(rowNames) {
			rowName = rowNames[rowIndex]
		}
		// Print row name with proper alignment
		rowLabel := runewidth.FillRight(utils.TruncateString(rowName, maxRowNameWidth), maxRowNameWidth+2)
		fmt.Print(colorText("1;37", rowLabel))

		for _, colIndex := range colIndices {
			// Determine cell value and type
			var rawValue any
			if rowIndex < len(dataMap[colIndex]) {
				rawValue = dataMap[colIndex][rowIndex]
			}
			value := "nil"
			if rawValue != nil {
				value = utils.FormatValue(rawValue)
			}

			// Choose color based on value type
			var valueColorCode string
			if rawValue == nil {
				valueColorCode = "2;37"
			} else {
				switch rawValue.(type) {
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
					valueColorCode = "0;34"
				case string:
					valueColorCode = "0;32"
				case bool:
					valueColorCode = "0;35"
				}
			}

			// Print cell with proper alignment using runewidth
			cellText := runewidth.FillRight(utils.TruncateString(value, colWidths[colIndex]), colWidths[colIndex]+1)
			if valueColorCode != "" {
				fmt.Print(" " + colorText(valueColorCode, cellText))
			} else {
				fmt.Print(" " + cellText)
			}
		}
		fmt.Println()
	}
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
func (dt *DataTable) ShowTypesRange(startEnd ...any) {
	dt.AtomicDo(func(dt *DataTable) {

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
				fmt.Println(colorText("2;37", "(empty range)"))
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
		fmt.Printf("%s %s\n", colorText("1;36", tableTitle), colorText("3;36", tableSummary))
		fmt.Println(strings.Repeat("=", min(width, 80)))

		// Handle empty table
		if rowCount == 0 || colCount == 0 {
			fmt.Println(colorText("3;36", "(empty)"))
			return
		}

		// Compute layout for type display (column widths, row names, and max row name width)
		colWidths, rowNames, maxRowNameWidth := prepareTableLayoutTypes(dt, dataMap, colIndices)

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
				fmt.Println("\n" + colorText("1;35", "--- Continue Type Display ---"))
			}
			if totalPages > 1 {
				pageInfo := fmt.Sprintf("--- Type Page %d/%d ---", page+1, totalPages)
				fmt.Println(colorText("1;36", pageInfo))

				// Display page navigation prompt
				if page > 0 && page < totalPages-1 {
					fmt.Println("(Scroll screen to see more)")
				}
			}

			// Print column names with runewidth alignment
			fmt.Print(colorText("1;32", runewidth.FillRight("RowNames", maxRowNameWidth+2)))
			for _, colIndex := range currentPageCols {
				lbl := utils.TruncateString(colIndex, colWidths[colIndex])
				fmt.Print(" " + colorText("1;32", runewidth.FillRight(lbl, colWidths[colIndex]+1)))
			}
			fmt.Println()

			// Print separator aligned to header widths
			fmt.Print(strings.Repeat("-", maxRowNameWidth+2))
			for _, colIndex := range currentPageCols {
				fmt.Print(" " + strings.Repeat("-", colWidths[colIndex]+1))
			}
			fmt.Println()
			// Print row data for the specified range
			selectedRowCount := end - start

			// Check if range was explicitly specified
			explicitRangeSpecified := len(startEnd) > 0

			// If there are too many rows in the selected range, only show first 20 and last 5
			// UNLESS a range was explicitly specified by the user
			if selectedRowCount > 25 && !explicitRangeSpecified {
				// Show first 20 rows
				printTypeRows(dataMap, start, start+20, rowNames, maxRowNameWidth, currentPageCols, colWidths)

				// Show ellipsis line aligned to columns
				fmt.Print(colorText("1;36", runewidth.FillRight("...", maxRowNameWidth+2)))
				for _, colIndex := range currentPageCols {
					fmt.Print(" " + colorText("1;36", runewidth.FillRight("...", colWidths[colIndex]+1)))
				}
				fmt.Println()

				// Show last 5 rows
				printTypeRows(dataMap, end-5, end, rowNames, maxRowNameWidth, currentPageCols, colWidths)

				// Show data summary
				fmt.Printf("\n%s\n", colorText("3;36", fmt.Sprintf("Displaying %d rows (from row %d to row %d)",
					selectedRowCount, start, end-1)))
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
	})
	fmt.Println()
}

// Print specified range of rows (type information)
func printTypeRows(dataMap map[string][]any, start, end int, rowNames []string, maxRowNameWidth int, colIndices []string, colWidths map[string]int) {
	for rowIndex := start; rowIndex < end; rowIndex++ {
		rowName := ""
		if rowIndex < len(rowNames) {
			rowName = rowNames[rowIndex]
		}
		// Use light gray color for row names with proper alignment using runewidth
		rowLabel := runewidth.FillRight(utils.TruncateString(rowName, maxRowNameWidth), maxRowNameWidth+2)
		fmt.Print(colorText("1;37", rowLabel))

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

			// Use different colors to distinguish types
			var typeColorCode string
			if value == "nil" {
				typeColorCode = "2;37"
			} else if strings.Contains(value, "int") || strings.Contains(value, "float") {
				typeColorCode = "0;34"
			} else if strings.Contains(value, "string") {
				typeColorCode = "0;32"
			} else if strings.Contains(value, "bool") {
				typeColorCode = "0;35"
			} else if strings.Contains(value, "map") || strings.Contains(value, "struct") {
				typeColorCode = "0;33"
			} else if strings.Contains(value, "slice") || strings.Contains(value, "array") || strings.Contains(value, "[]") {
				typeColorCode = "0;36"
			}

			// Print type cell with proper alignment using runewidth
			cellText := runewidth.FillRight(utils.TruncateString(value, colWidths[colIndex]), colWidths[colIndex]+1)
			if typeColorCode != "" {
				fmt.Print(" " + colorText(typeColorCode, cellText))
			} else {
				fmt.Print(" " + cellText)
			}
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
		fmt.Println(colorText("1;31", "ERROR: Unable to show a nil DataList"))
		return
	}

	dl.AtomicDo(func(dl *DataList) {

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
				fmt.Printf("%s %s\n", colorText("1;33", dataTitle), colorText("3;33", fmt.Sprintf("(%d items)", totalItems)))
				fmt.Println(strings.Repeat("=", min(width, 80)))
				fmt.Println(colorText("2;37", "(empty range)"))
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
		fmt.Printf("%s %s\n", colorText("1;33", dataTitle), colorText("3;33", dataSummary))
		fmt.Println(strings.Repeat("=", min(width, 80)))

		// Check if DataList is empty
		if totalItems == 0 {
			fmt.Println(colorText("2;37", "(empty)"))
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
						statsLine := fmt.Sprintf(" %s: mean=%.2f, min=%.2f, max=%.2f, range=%.2f",
							statsLabel, mean, dlmin, max, max-dlmin)
						fmt.Println(colorText("3;33", statsLine))
						if len(numericValues) > 10 {
							// Show sd and median with two decimal places
							fmt.Println(colorText("3;33", fmt.Sprintf("      sd=%.2f, median=%.2f",
								rangeDl.Stdev(), rangeDl.Median())))
						}
						fmt.Println(strings.Repeat("-", min(width, 80)))
					}
				}
			}
		}

		// Always show in linear format regardless of terminal width
		fmt.Printf("%s  %s\n", colorText("1;32", "Index"), colorText("1;32", "Value"))
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
				strValue := utils.FormatValue(value)

				// Determine color based on value type
				var valueColorCode string
				if value == nil {
					valueColorCode = "2;37"
				} else {
					switch value.(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
						valueColorCode = "0;34"
					case string:
						valueColorCode = "0;32"
					case bool:
						valueColorCode = "0;35"
					}
				}

				indexText := colorText("1;37", fmt.Sprintf("%-6d", itemIndex))
				valueText := strValue
				if valueColorCode != "" {
					valueText = colorText(valueColorCode, strValue)
				}
				fmt.Printf("%s %s\n", indexText, valueText)
			}
		}

		// If there are too many items in the range, show ellipsis and the last few items
		if !showAll {
			fmt.Println(colorText("1;33", "...    ..."))

			// Show last 5 items from the range
			for i := end - 5; i < end; i++ {
				value := dl.data[i]
				strValue := utils.FormatValue(value)

				// Color based on value type
				var valueColorCode string
				if value == nil {
					valueColorCode = "2;37"
				} else {
					switch value.(type) {
					case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
						valueColorCode = "0;34"
					case string:
						valueColorCode = "0;32"
					case bool:
						valueColorCode = "0;35"
					}
				}

				indexText := colorText("1;37", fmt.Sprintf("%-6d", i))
				valueText := strValue
				if valueColorCode != "" {
					valueText = colorText(valueColorCode, strValue)
				}
				fmt.Printf("%s %s\n", indexText, valueText)
			}

			// Show data summary
			fmt.Printf("\n%s\n", colorText("3;33", fmt.Sprintf("Displaying %d items (from index %d to index %d)",
				selectedItems, start, end-1)))
		}
	})
	fmt.Println()
}

// ShowTypes displays the data types of each element in the DataList.
// It adapts to terminal width and always displays in a linear format.
func (dl *DataList) ShowTypes() {
	dl.ShowTypesRange()
}

// ShowTypesRange displays the data types of DataList within a specified range.
// startEnd 参数同 ShowRange。
func (dl *DataList) ShowTypesRange(startEnd ...any) {
	if dl == nil {
		fmt.Println(colorText("1;31", "ERROR: Unable to show types of a nil DataList"))
		return
	}
	dl.AtomicDo(func(dl *DataList) {

		// 取 terminal 寬度
		width := getDataListTerminalWidth()
		// 標題
		title := "DataList Type Info"
		if dl.name != "" {
			title += ": " + dl.name
		}

		// 計算顯示範圍
		total := len(dl.data)
		start, end := 0, total
		if len(startEnd) > 0 {
			if v, ok := startEnd[0].(int); ok {
				if v < 0 {
					start = max(0, total+v)
				} else {
					end = min(v, total)
				}
			}
			if len(startEnd) >= 2 {
				if v2 := startEnd[1]; v2 == nil {
					end = total
				} else if e, ok := v2.(int); ok {
					if e < 0 {
						end = total + e
					} else {
						end = min(e, total)
					}
				}
			}
			if start < 0 {
				start = 0
			}
			if end > total {
				end = total
			}
			if start >= end {
				fmt.Printf("%s %s\n", colorText("1;33", title), colorText("3;33", fmt.Sprintf("(%d items)", total)))
				fmt.Println(strings.Repeat("=", min(width, 80)))
				fmt.Println(colorText("2;37", "(empty range)"))
				return
			}
		}

		// 標題列
		summary := fmt.Sprintf("(%d items)", total)
		if len(startEnd) > 0 {
			summary += fmt.Sprintf(" [showing items %d to %d]", start, end-1)
		}
		fmt.Printf("%s %s\n", colorText("1;33", title), colorText("3;33", summary))
		fmt.Println(strings.Repeat("=", min(width, 80)))

		// 無資料
		if end-start == 0 {
			fmt.Println(colorText("2;37", "(empty)"))
			return
		}

		// 計算 Index 與 Type 欄位最大寬度
		maxIdxW := runewidth.StringWidth("Index")
		maxTypW := runewidth.StringWidth("Type")
		types := make([]string, end-start)
		for i := start; i < end; i++ {
			idx := strconv.Itoa(i)
			if w := runewidth.StringWidth(idx); w > maxIdxW {
				maxIdxW = w
			}
			var tstr string
			if dl.data[i] == nil {
				tstr = "nil"
			} else {
				tstr = reflect.TypeOf(dl.data[i]).String()
			}
			types[i-start] = tstr
			if w := runewidth.StringWidth(tstr); w > maxTypW {
				maxTypW = w
			}
		}

		// 列印表頭
		fmt.Print(colorText("1;32", runewidth.FillRight("Index", maxIdxW+2)))
		fmt.Println(" " + colorText("1;32", runewidth.FillRight("Type", maxTypW+1)))
		// 列印分隔線
		fmt.Println(strings.Repeat("-", maxIdxW+2+1+maxTypW+1))

		// 計算範圍內項目數
		totalItems := end - start
		explicit := len(startEnd) > 0
		const maxShow = 25
		showAll := explicit || totalItems <= maxShow
		firstCount := totalItems
		if !showAll {
			firstCount = 20
		}

		// 列印前幾項
		for i := start; i < start+firstCount; i++ {
			idx := strconv.Itoa(i)
			tstr := types[i-start]
			fmt.Print(colorText("1;37", runewidth.FillRight(idx, maxIdxW+2)))
			fmt.Println(" " + runewidth.FillRight(tstr, maxTypW+1))
		}

		// 如果有截斷，則列印省略號及最後幾項
		if !showAll {
			// ellipsis line
			fmt.Print(colorText("1;33", runewidth.FillRight("...", maxIdxW+2)))
			fmt.Println(" " + colorText("1;33", runewidth.FillRight("...", maxTypW+1)))
			// last 5 items
			for i := end - 5; i < end; i++ {
				idx := strconv.Itoa(i)
				tstr := types[i-start]
				fmt.Print(colorText("1;37", runewidth.FillRight(idx, maxIdxW+2)))
				fmt.Println(" " + runewidth.FillRight(tstr, maxTypW+1))
			}
			// summary
			fmt.Printf("\n%s\n", colorText("3;33", fmt.Sprintf("Displaying %d items (from index %d to index %d)",
				totalItems, start, end-1)))
		}
	})
	fmt.Println()
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

// prepareTableLayout 计算每个列的宽度、行名列表及最大行名宽度
func prepareTableLayout(dt *DataTable, dataMap map[string][]any, colIndices []string) (map[string]int, []string, int) {
	// 计算每个列的最大宽度
	colWidths := make(map[string]int, len(colIndices))
	for _, idx := range colIndices {
		width := runewidth.StringWidth(idx)
		for _, v := range dataMap[idx] {
			s := utils.FormatValue(v)
			if w := runewidth.StringWidth(s); w > width {
				width = w
			}
		}
		// 限制列宽不超过30
		if width > 30 {
			width = 30
		}
		colWidths[idx] = width
	}
	// 计算行名及最大行名宽度
	totalRows := dt.getMaxColLength()
	rowNames := make([]string, totalRows)
	maxRowName := runewidth.StringWidth("RowNames")
	for i := 0; i < totalRows; i++ {
		name, _ := dt.getRowNameByIndex(i)
		label := fmt.Sprintf("%d: %s", i, name)
		rowNames[i] = label
		if w := runewidth.StringWidth(label); w > maxRowName {
			maxRowName = w
		}
	}
	// 限制行名宽度不超过25
	if maxRowName > 25 {
		maxRowName = 25
	}
	return colWidths, rowNames, maxRowName
}

// prepareTableLayoutTypes 计算 ShowTypesRange 使用的列宽、行名列表及最大行名宽度
func prepareTableLayoutTypes(dt *DataTable, dataMap map[string][]any, colIndices []string) (map[string]int, []string, int) {
	// 计算每个列的最大宽度（以类型字符串宽度为根据）
	colWidths := make(map[string]int, len(colIndices))
	for _, idx := range colIndices {
		// 初始宽度为列名宽度
		width := runewidth.StringWidth(idx)
		// 遍历每个单元格，计算类型字符串或特殊标记宽度
		for _, v := range dataMap[idx] {
			var s string
			if v == nil {
				s = "nil"
			} else {
				// 获取类型字符串或特殊描述
				switch val := v.(type) {
				case []any:
					s = fmt.Sprintf("[]any(len=%d)", len(val))
				case []string:
					s = fmt.Sprintf("[]string(len=%d)", len(val))
				case map[string]any:
					s = fmt.Sprintf("map[string]any(len=%d)", len(val))
				case time.Time:
					s = "time.Time"
				default:
					s = reflect.TypeOf(v).String()
				}
			}
			if w := runewidth.StringWidth(s); w > width {
				width = w
			}
		}
		// 限制列宽不超过25
		if width > 25 {
			width = 25
		}
		colWidths[idx] = width
	}
	// 计算行名及最大行名宽度
	total := dt.getMaxColLength()
	rowNames := make([]string, total)
	maxName := runewidth.StringWidth("RowNames")
	for i := 0; i < total; i++ {
		name, _ := dt.getRowNameByIndex(i)
		label := fmt.Sprintf("%d: %s", i, name)
		rowNames[i] = label
		if w := runewidth.StringWidth(label); w > maxName {
			maxName = w
		}
	}
	if maxName > 25 {
		maxName = 25
	}
	return colWidths, rowNames, maxName
}
