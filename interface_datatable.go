package insyra

import "gorm.io/gorm"

// IDataTable defines the behavior expected from a DataTable.
type IDataTable interface {
	AppendCols(columns ...*DataList) *DataTable
	AppendRowsFromDataList(rowsData ...*DataList) *DataTable
	AppendRowsByColIndex(rowsData ...map[string]any) *DataTable
	AppendRowsByColName(rowsData ...map[string]any) *DataTable
	GetElement(rowIndex int, columnIndex string) any
	GetElementByNumberIndex(rowIndex int, columnIndex int) any
	GetCol(index string) *DataList
	GetColByNumber(index int) *DataList
	GetColByName(name string) *DataList
	GetRow(index int) *DataList
	GetRowByName(name string) *DataList // Add this line
	UpdateElement(rowIndex int, columnIndex string, value any)
	UpdateCol(index string, dl *DataList)
	UpdateColByNumber(index int, dl *DataList)
	UpdateRow(index int, dl *DataList)
	SetColToRowNames(columnIndex string) *DataTable
	SetRowToColNames(rowIndex int) *DataTable
	ChangeColName(oldName, newName string) *DataTable
	GetColNameByNumber(index int) string
	SetColNameByIndex(index string, name string) *DataTable
	SetColNameByNumber(numberIndex int, name string) *DataTable
	ColNamesToFirstRow() *DataTable
	DropColNames() *DataTable
	FindRowsIfContains(value any) []int
	FindRowsIfContainsAll(values ...any) []int
	FindRowsIfAnyElementContainsSubstring(substring string) []int
	FindRowsIfAllElementsContainSubstring(substring string) []int
	FindColsIfContains(value any) []string
	FindColsIfContainsAll(values ...any) []string
	FindColsIfAnyElementContainsSubstring(substring string) []string
	FindColsIfAllElementsContainSubstring(substring string) []string
	DropColsByName(columnNames ...string)
	DropColsByIndex(columnIndices ...string)
	DropColsByNumber(columnIndices ...int)
	DropColsContainStringElements()
	DropColsContainNumbers()
	DropColsContainNil()
	DropRowsByIndex(rowIndices ...int)
	DropRowsByName(rowNames ...string)
	DropRowsContainStringElements()
	DropRowsContainNumbers()
	DropRowsContainNil()
	Data(useNamesAsKeys ...bool) map[string][]any
	Show()
	ShowTypes()
	ShowRange(startEnd ...any)
	ShowTypesRange(startEnd ...any)
	GetRowNameByIndex(index int) string
	SetRowNameByIndex(index int, name string)
	ChangeRowName(oldName, newName string) *DataTable
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	getRowNameByIndex(index int) (string, bool)
	getMaxColLength() int
	updateTimestamp()

	// name
	GetName() string
	SetName(name string) *DataTable

	// Statistics
	Size() (numRows int, numCols int)
	Count(value any) int
	Mean() any
	Summary() // Conversion
	Transpose() *DataTable
	Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable

	// Filters
	Filter(filterFunc FilterFunc) *DataTable
	FilterByCustomElement(f func(value any) bool) *DataTable
	FilterByColIndexGreaterThan(threshold string) *DataTable
	FilterByColIndexGreaterThanOrEqualTo(threshold string) *DataTable
	FilterByColIndexLessThan(threshold string) *DataTable
	FilterByColIndexLessThanOrEqualTo(threshold string) *DataTable
	FilterByColIndexEqualTo(index string) *DataTable
	FilterByColNameEqualTo(name string) *DataTable
	FilterByColNameContains(substring string) *DataTable
	FilterByRowNameEqualTo(name string) *DataTable
	FilterByRowNameContains(substring string) *DataTable
	FilterByRowIndexGreaterThan(threshold int) *DataTable
	FilterByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable
	FilterByRowIndexLessThan(threshold int) *DataTable
	FilterByRowIndexLessThanOrEqualTo(threshold int) *DataTable
	FilterByRowIndexEqualTo(index int) *DataTable

	// Swap
	SwapColsByName(columnName1 string, columnName2 string) *DataTable
	SwapColsByIndex(columnIndex1 string, columnIndex2 string) *DataTable
	SwapColsByNumber(columnNumber1 int, columnNumber2 int) *DataTable
	SwapRowsByIndex(rowIndex1 int, rowIndex2 int) *DataTable
	SwapRowsByName(rowName1 string, rowName2 string) *DataTable

	// CSV
	ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error
	LoadFromCSV(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool) error

	// JSON
	ToJSON(filePath string, useColName bool) error
	ToJSON_Bytes(useColName bool) []byte
	LoadFromJSON(filePath string) error
	LoadFromJSON_Bytes(jsonData []byte) error

	ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error

	sortColsByIndex()
	regenerateColIndex()
}
