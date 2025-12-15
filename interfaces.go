package insyra

import "gorm.io/gorm"

// IDataList defines the behavior expected from a DataList.
type IDataList interface {
	AtomicDo(func(*DataList))
	GetCreationTimestamp() int64
	GetLastModifiedTimestamp() int64
	updateTimestamp()
	GetName() string
	SetName(string) *DataList
	Data() []any
	Append(values ...any)
	Get(index int) any
	Clone() *DataList
	Count(value any) int
	Counter() map[any]int
	Update(index int, value any)
	InsertAt(index int, value any)
	FindFirst(any) any
	FindLast(any) any
	FindAll(any) []int
	Filter(func(any) bool) *DataList
	ReplaceFirst(any, any)
	ReplaceLast(any, any)
	ReplaceAll(any, any)
	ReplaceOutliers(float64, float64) *DataList
	Pop() any
	Drop(index int) *DataList
	DropAll(...any) *DataList
	DropIfContains(any) *DataList
	Clear() *DataList
	ClearStrings() *DataList
	ClearNumbers() *DataList
	ClearNaNs() *DataList
	ClearOutliers(float64) *DataList
	Normalize() *DataList
	Standardize() *DataList
	FillNaNWithMean() *DataList
	MovingAverage(int) *DataList
	WeightedMovingAverage(int, any) *DataList
	ExponentialSmoothing(float64) *DataList
	DoubleExponentialSmoothing(float64, float64) *DataList
	MovingStdev(int) *DataList
	Len() int
	Sort(ascending ...bool) *DataList
	Map(mapFunc func(int, any) any) *DataList
	Rank() *DataList
	Reverse() *DataList
	Upper() *DataList
	Lower() *DataList
	Capitalize() *DataList // Statistics
	Sum() float64
	Max() float64
	Min() float64
	Mean() float64
	WeightedMean(weights any) float64
	GMean() float64
	Median() float64
	Mode() []float64
	MAD() float64
	Stdev() float64
	StdevP() float64
	Var() float64
	VarP() float64
	Range() float64
	Quartile(int) float64
	IQR() float64
	Percentile(float64) float64
	Difference() *DataList
	Summary()

	// comparison
	IsEqualTo(*DataList) bool
	IsTheSameAs(*DataList) bool
	Show()
	ShowRange(startEnd ...any)
	ShowTypes()
	ShowTypesRange(startEnd ...any)

	// conversion
	ParseNumbers() *DataList
	ParseStrings() *DataList
	ToF64Slice() []float64
	ToStringSlice() []string

	// Interpolation
	LinearInterpolation(float64) float64
	QuadraticInterpolation(float64) float64
	LagrangeInterpolation(float64) float64
	NearestNeighborInterpolation(float64) float64
	NewtonInterpolation(float64) float64
	HermiteInterpolation(float64, []float64) float64
}

// IDataTable defines the behavior expected from a DataTable.
type IDataTable interface {
	AtomicDo(func(*DataTable))
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
	GetRowByName(name string) *DataList
	UpdateElement(rowIndex int, columnIndex string, value any)
	UpdateCol(index string, dl *DataList)
	UpdateColByNumber(index int, dl *DataList)
	UpdateRow(index int, dl *DataList)
	SetColToRowNames(columnIndex string) *DataTable
	SetRowToColNames(rowIndex int) *DataTable
	ChangeColName(oldName, newName string) *DataTable
	GetColNameByNumber(index int) string
	GetColIndexByName(name string) string
	GetColIndexByNumber(number int) string
	GetColNumberByName(name string) int
	SetColNameByIndex(index string, name string) *DataTable
	SetColNameByNumber(numberIndex int, name string) *DataTable
	ColNamesToFirstRow() *DataTable
	DropColNames() *DataTable
	ColNames() []string
	Headers() []string
	SetColNames(colNames []string) *DataTable
	SetHeaders(headers []string) *DataTable
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
	DropColsContain(value ...any)
	DropColsContainExcelNA()
	DropRowsByIndex(rowIndices ...int)
	DropRowsByName(rowNames ...string)
	DropRowsContainStringElements()
	DropRowsContainNumbers()
	DropRowsContainNil()
	DropRowsContain(value ...any)
	DropRowsContainExcelNA()
	Data(useNamesAsKeys ...bool) map[string][]any
	// ToMap is the alias for Data().
	// It returns a map[string][]any representation of the DataTable.
	// Parameters:
	// - useNamesAsKeys: Whether to use column names as keys in the returned map
	// Returns:
	// - map[string][]any: DataTable represented as a map of columns
	ToMap(useNamesAsKeys ...bool) map[string][]any
	Show()
	ShowTypes()
	ShowRange(startEnd ...any)
	ShowTypesRange(startEnd ...any)
	GetRowNameByIndex(index int) string
	SetRowNameByIndex(index int, name string)
	ChangeRowName(oldName, newName string) *DataTable
	RowNamesToFirstCol() *DataTable
	DropRowNames() *DataTable
	RowNames() []string
	SetRowNames(rowNames []string) *DataTable
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
	Summary()

	// Operations
	Transpose() *DataTable
	Clone() *DataTable
	To2DSlice() [][]any
	SimpleRandomSample(sampleSize int) *DataTable
	Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable
	SortBy(configs ...DataTableSortConfig) *DataTable

	// Filters
	Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable
	FilterByCustomElement(f func(value any) bool) *DataTable
	FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable
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
	// JSON
	// ToJSON saves the DataTable as a JSON file.
	// Parameters:
	// - filePath: Output JSON file path
	// - useColNames: Whether to use column names as keys in JSON objects
	// Returns:
	// - error: Error information, returns nil if successful
	ToJSON(filePath string, useColNames bool) error
	// ToJSON_Bytes converts the DataTable to JSON format and returns it as a byte slice.
	// Parameters:
	// - useColNames: Whether to use column names as keys in JSON objects
	// Returns:
	// - []byte: JSON data as byte slice
	ToJSON_Bytes(useColNames bool) []byte
	// ToJSON_String converts the DataTable to JSON format and returns it as a string.
	// Parameters:
	// - useColNames: Whether to use column names as keys in JSON objects
	// Returns:
	// - string: JSON data as a string
	ToJSON_String(useColNames bool) string

	ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error

	AddColUsingCCL(newColName, ccl string) *DataTable

	sortColsByIndex()
	regenerateColIndex()
}
