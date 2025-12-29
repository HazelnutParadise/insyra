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
	Concat(other IDataList) *DataList
	AppendDataList(other IDataList) *DataList
	Get(index int) any
	Clone() *DataList
	Count(value any) int
	Counter() map[any]int
	Update(index int, value any) *DataList
	InsertAt(index int, value any) *DataList
	FindFirst(any) any
	FindLast(any) any
	FindAll(any) []int
	Filter(func(any) bool) *DataList
	ReplaceFirst(any, any) *DataList
	ReplaceLast(any, any) *DataList
	ReplaceAll(any, any) *DataList
	ReplaceOutliers(float64, float64) *DataList
	Pop() any
	Drop(index int) *DataList
	DropAll(...any) *DataList
	DropIfContains(string) *DataList
	Clear() *DataList
	ClearStrings() *DataList
	ClearNumbers() *DataList
	ClearNaNs() *DataList
	ClearNils() *DataList
	ClearNilsAndNaNs() *DataList
	ClearOutliers(float64) *DataList
	ReplaceNaNsWith(any) *DataList
	ReplaceNilsWith(any) *DataList
	ReplaceNaNsAndNilsWith(any) *DataList
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
	UpdateElement(rowIndex int, columnIndex string, value any) *DataTable
	UpdateCol(index string, dl *DataList) *DataTable
	UpdateColByNumber(index int, dl *DataList) *DataTable
	UpdateRow(index int, dl *DataList) *DataTable
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
	DropColsByName(columnNames ...string) *DataTable
	DropColsByIndex(columnIndices ...string) *DataTable
	DropColsByNumber(columnIndices ...int) *DataTable
	DropColsContainString() *DataTable
	DropColsContainNumber() *DataTable
	DropColsContainNil() *DataTable
	DropColsContainNaN() *DataTable
	DropColsContain(value ...any) *DataTable
	DropColsContainExcelNA() *DataTable
	DropRowsByIndex(rowIndices ...int) *DataTable
	DropRowsByName(rowNames ...string) *DataTable
	DropRowsContainString() *DataTable
	DropRowsContainNumber() *DataTable
	DropRowsContainNil() *DataTable
	DropRowsContainNaN() *DataTable
	DropRowsContain(value ...any) *DataTable
	DropRowsContainExcelNA() *DataTable
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
	// GetRowIndexByName returns the index of a row by its name.
	// Returns -1 and false if the row name does not exist.
	// Always check the boolean return value to distinguish between "name not found" and "last row",
	// since -1 typically represents the last element in Insyra's Get methods.
	GetRowIndexByName(name string) (int, bool)
	// GetRowNameByIndex returns the name of a row at the given index.
	// Returns empty string and false if no name is set for the row.
	GetRowNameByIndex(index int) (string, bool)
	SetRowNameByIndex(index int, name string) *DataTable
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
	NumRows() int
	NumCols() int
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
	FilterCols(filterFunc func(rowIndex int, rowName string, x any) bool) *DataTable
	FilterColsByColIndexGreaterThan(threshold string) *DataTable
	FilterColsByColIndexGreaterThanOrEqualTo(threshold string) *DataTable
	FilterColsByColIndexLessThan(threshold string) *DataTable
	FilterColsByColIndexLessThanOrEqualTo(threshold string) *DataTable
	FilterColsByColIndexEqualTo(index string) *DataTable
	FilterColsByColNameEqualTo(name string) *DataTable
	FilterColsByColNameContains(substring string) *DataTable
	FilterRowsByRowNameEqualTo(name string) *DataTable
	FilterRowsByRowNameContains(substring string) *DataTable
	FilterRowsByRowIndexGreaterThan(threshold int) *DataTable
	FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable
	FilterRowsByRowIndexLessThan(threshold int) *DataTable
	FilterRowsByRowIndexLessThanOrEqualTo(threshold int) *DataTable
	FilterRowsByRowIndexEqualTo(index int) *DataTable

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

	Merge(other IDataTable, direction MergeDirection, mode MergeMode, on ...string) (*DataTable, error)

	AddColUsingCCL(newColName, ccl string) *DataTable

	// Replace
	Replace(oldValue, newValue any) *DataTable
	ReplaceNaNsWith(newValue any) *DataTable
	ReplaceNilsWith(newValue any) *DataTable
	ReplaceNaNsAndNilsWith(newValue any) *DataTable
	ReplaceInRow(rowIndex int, oldValue, newValue any, mode ...int) *DataTable
	ReplaceNaNsInRow(rowIndex int, newValue any, mode ...int) *DataTable
	ReplaceNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable
	ReplaceNaNsAndNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable
	ReplaceInCol(colIndex string, oldValue, newValue any, mode ...int) *DataTable
	ReplaceNaNsInCol(colIndex string, newValue any, mode ...int) *DataTable
	ReplaceNilsInCol(colIndex string, newValue any, mode ...int) *DataTable
	ReplaceNaNsAndNilsInCol(colIndex string, newValue any, mode ...int) *DataTable

	sortColsByIndex()
	regenerateColIndex()
}
