package ccl

// Context defines the interface for data access in CCL.
// Implement this interface to allow CCL to operate on your custom data structures.
type Context interface {
	// GetCol returns the value of the column at the given index for the current row.
	GetCol(index int) any

	// GetColByName returns the value of the column with the given name for the current row.
	GetColByName(name string) (any, error)

	// GetRowIndex returns the index of the current row.
	GetRowIndex() int

	// GetCurrentRow returns the current row data (e.g., as []any or map).
	// This is used for the @ operator.
	GetCurrentRow() any

	// GetCell returns the value at the specified column index and row index.
	// Used for the . operator (e.g., A.prev).
	GetCell(colIndex, rowIndex int) (any, error)

	// GetCellByName returns the value at the specified column name and row index.
	// Used for the . operator (e.g., 'Col'.prev).
	GetCellByName(colName string, rowIndex int) (any, error)

	// GetRowAt returns the row data at the specified row index.
	// Used for @.prev.
	GetRowAt(rowIndex int) (any, error)

	// GetRowIndexByName returns the index of the row with the given name.
	// Used for row name references in the . operator.
	GetRowIndexByName(rowName string) (int, error)

	// GetColData returns the entire data of the column at the given index.
	// Used for aggregate functions.
	GetColData(index int) ([]any, error)

	// GetColDataByName returns the entire data of the column with the given name.
	// Used for aggregate functions.
	GetColDataByName(name string) ([]any, error)

	// GetRowCount returns the total number of rows in the context.
	GetRowCount() int

	// SetRowIndex sets the current row index for the context.
	// This is used for iterating over rows during aggregation of complex expressions.
	SetRowIndex(index int) error

	// GetAllData returns all data in the context as a single flattened slice.
	// This is used for the @ operator in aggregate functions (e.g., SUM(@)).
	GetAllData() ([]any, error)
}
