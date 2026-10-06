package ccl

import (
	"errors"

	internalccl "github.com/HazelnutParadise/insyra/internal/ccl"
)

// MapContext is a Context over columns held in a map, for evaluating CCL
// without a DataTable. Make one with NewMapContext; columns are ordered by name,
// so A is the alphabetically first column. Move between rows with SetRowIndex,
// which refuses a row that is not there. The zero MapContext holds no columns
// and refuses every read with an error.
type MapContext struct {
	m *internalccl.MapContext
}

var errNoMapContext = errors.New("ccl: the MapContext is empty; make one with NewMapContext")

// NewMapContext creates a MapContext over data, a map from column name to the
// column's values. Every column must have the same length.
func NewMapContext(data map[string][]any) (*MapContext, error) {
	m, err := internalccl.NewMapContext(data)
	if err != nil {
		return nil, err
	}
	return &MapContext{m: m}, nil
}

// GetCol returns the value of the column at index for the current row.
func (c *MapContext) GetCol(index int) any {
	if c == nil || c.m == nil {
		return nil
	}
	return c.m.GetCol(index)
}

// GetColByName returns the value of the named column for the current row.
func (c *MapContext) GetColByName(name string) (any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetColByName(name)
}

// GetRowIndex returns the current row.
func (c *MapContext) GetRowIndex() int {
	if c == nil || c.m == nil {
		return 0
	}
	return c.m.GetRowIndex()
}

// GetCurrentRow returns the current row's values, for the @ operator.
func (c *MapContext) GetCurrentRow() any {
	if c == nil || c.m == nil {
		return nil
	}
	return c.m.GetCurrentRow()
}

// GetCell returns the value at a column index and a row.
func (c *MapContext) GetCell(colIndex, rowIndex int) (any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetCell(colIndex, rowIndex)
}

// GetCellByName returns the value at a column name and a row.
func (c *MapContext) GetCellByName(colName string, rowIndex int) (any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetCellByName(colName, rowIndex)
}

// GetRowAt returns the values of a row.
func (c *MapContext) GetRowAt(rowIndex int) (any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetRowAt(rowIndex)
}

// GetRowIndexByName returns the index of a named row. A MapContext has no row
// names, so it always returns an error.
func (c *MapContext) GetRowIndexByName(rowName string) (int, error) {
	if c == nil || c.m == nil {
		return -1, errNoMapContext
	}
	return c.m.GetRowIndexByName(rowName)
}

// GetColData returns the whole column at index.
func (c *MapContext) GetColData(index int) ([]any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetColData(index)
}

// GetColDataByName returns the whole named column.
func (c *MapContext) GetColDataByName(name string) ([]any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetColDataByName(name)
}

// GetColIndexByName returns the index of the named column.
func (c *MapContext) GetColIndexByName(colName string) (int, error) {
	if c == nil || c.m == nil {
		return -1, errNoMapContext
	}
	return c.m.GetColIndexByName(colName)
}

// GetColCount returns the number of columns.
func (c *MapContext) GetColCount() int {
	if c == nil || c.m == nil {
		return 0
	}
	return c.m.GetColCount()
}

// GetRowCount returns the number of rows.
func (c *MapContext) GetRowCount() int {
	if c == nil || c.m == nil {
		return 0
	}
	return c.m.GetRowCount()
}

// SetRowIndex moves to a row, refusing one that is not there.
func (c *MapContext) SetRowIndex(index int) error {
	if c == nil || c.m == nil {
		return errNoMapContext
	}
	return c.m.SetRowIndex(index)
}

// GetAllData returns every value, for @ inside an aggregate function.
func (c *MapContext) GetAllData() ([]any, error) {
	if c == nil || c.m == nil {
		return nil, errNoMapContext
	}
	return c.m.GetAllData()
}
