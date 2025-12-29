package ccl

import (
	"fmt"
)

// MapContext implements Context for a map of columns (map[string][]any).
// This allows using CCL with native Go maps and slices.
type MapContext struct {
	Data          map[string][]any
	Rows          int
	CurrentRowIdx int
	ColNameMap    map[string]int // Optional: if we want to support index access, we need an order
	ColNames      []string       // To support index access
}

// NewMapContext creates a new MapContext from a map of columns.
func NewMapContext(data map[string][]any) (*MapContext, error) {
	rows := 0
	colNames := make([]string, 0, len(data))
	colNameMap := make(map[string]int, len(data))

	i := 0
	for name, col := range data {
		if rows == 0 {
			rows = len(col)
		} else if len(col) != rows {
			return nil, fmt.Errorf("column lengths mismatch: column '%s' has length %d, expected %d", name, len(col), rows)
		}
		colNames = append(colNames, name)
		colNameMap[name] = i
		i++
	}

	return &MapContext{
		Data:       data,
		Rows:       rows,
		ColNameMap: colNameMap,
		ColNames:   colNames,
	}, nil
}

func (c *MapContext) GetCol(index int) any {
	if index < 0 || index >= len(c.ColNames) {
		return nil
	}
	name := c.ColNames[index]
	col := c.Data[name]
	if c.CurrentRowIdx < 0 || c.CurrentRowIdx >= len(col) {
		return nil
	}
	return col[c.CurrentRowIdx]
}

func (c *MapContext) GetColByName(name string) (any, error) {
	col, ok := c.Data[name]
	if !ok {
		return nil, fmt.Errorf("column '%s' not found", name)
	}
	if c.CurrentRowIdx < 0 || c.CurrentRowIdx >= len(col) {
		return nil, fmt.Errorf("row index %d out of range", c.CurrentRowIdx)
	}
	return col[c.CurrentRowIdx], nil
}

func (c *MapContext) GetRowIndex() int {
	return c.CurrentRowIdx
}

func (c *MapContext) GetCurrentRow() any {
	// Construct row map
	row := make(map[string]any, len(c.Data))
	for name, col := range c.Data {
		if c.CurrentRowIdx < len(col) {
			row[name] = col[c.CurrentRowIdx]
		}
	}
	return row
}

func (c *MapContext) GetCell(colIndex, rowIndex int) (any, error) {
	if colIndex < 0 || colIndex >= len(c.ColNames) {
		return nil, fmt.Errorf("column index %d out of range", colIndex)
	}
	name := c.ColNames[colIndex]
	col := c.Data[name]
	if rowIndex < 0 || rowIndex >= len(col) {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}
	return col[rowIndex], nil
}

func (c *MapContext) GetCellByName(colName string, rowIndex int) (any, error) {
	col, ok := c.Data[colName]
	if !ok {
		return nil, fmt.Errorf("column '%s' not found", colName)
	}
	if rowIndex < 0 || rowIndex >= len(col) {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}
	return col[rowIndex], nil
}

func (c *MapContext) GetRowAt(rowIndex int) (any, error) {
	if rowIndex < 0 || rowIndex >= c.Rows {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}
	row := make(map[string]any, len(c.Data))
	for name, col := range c.Data {
		row[name] = col[rowIndex]
	}
	return row, nil
}

func (c *MapContext) GetRowIndexByName(rowName string) (int, error) {
	return -1, fmt.Errorf("row names not supported in MapContext")
}

func (c *MapContext) GetAllData() ([]any, error) {
	var allData []any
	totalSize := len(c.ColNames) * c.Rows
	allData = make([]any, 0, totalSize)

	for _, name := range c.ColNames {
		if col, ok := c.Data[name]; ok {
			allData = append(allData, col...)
		}
	}
	return allData, nil
}

func (c *MapContext) GetColData(index int) ([]any, error) {
	if index < 0 || index >= len(c.ColNames) {
		return nil, fmt.Errorf("column index %d out of range", index)
	}
	name := c.ColNames[index]
	return c.Data[name], nil
}

func (c *MapContext) GetColDataByName(name string) ([]any, error) {
	col, ok := c.Data[name]
	if !ok {
		return nil, fmt.Errorf("column '%s' not found", name)
	}
	return col, nil
}

func (c *MapContext) GetColIndexByName(colName string) (int, error) {
	idx, ok := c.ColNameMap[colName]
	if !ok {
		return -1, fmt.Errorf("column name '%s' not found", colName)
	}
	return idx, nil
}

func (c *MapContext) GetColCount() int {
	return len(c.ColNames)
}

func (c *MapContext) GetRowCount() int {
	return c.Rows
}

func (c *MapContext) SetRowIndex(index int) error {
	if index < 0 || index >= c.Rows {
		return fmt.Errorf("row index %d out of range", index)
	}
	c.CurrentRowIdx = index
	return nil
}
