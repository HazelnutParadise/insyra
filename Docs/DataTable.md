# DataTable

DataTable is a flexible and efficient data structure for managing two-dimensional data in Go. It provides a variety of methods for data manipulation, querying, and display. The DataTable structure uses a column-based storage model, which allows for efficient column operations and flexible row management.

## Structure

```go
type DataTable struct {
    mu                    sync.Mutex
    columns               []*DataList
    columnIndex           map[string]int
    rowNames              map[string]int
    creationTimestamp     int64
    lastModifiedTimestamp atomic.Int64
}
```

## Main Features

- Column-based storage for efficient column operations
- Support for named rows and columns
- Thread-safe operations
- Flexible data manipulation (append, update, delete)
- Advanced querying capabilities
- Timestamp tracking for creation and modification

## Constructor

#### NewDataTable

```go
func NewDataTable(columns ...*DataList) *DataTable
```

Creates a new DataTable with optional initial columns.

## Methods

##### `AppendCols(columns ...*DataList) *DataTable`
Appends columns to the `DataTable`. If the new columns are shorter than existing ones, `nil` values will be appended to match the length. If they are longer, existing columns will be extended with `nil` values.

##### `AppendRowsFromDataList(rowsData ...*DataList) *DataTable`
Appends rows to the `DataTable` using `DataList` objects. If the new rows are shorter than existing columns, `nil` values will be appended to match the length.

##### `AppendRowsByColIndex(rowsData ...map[string]interface{}) *DataTable`
Appends rows to the `DataTable` by mapping column indices to values. If necessary, new columns are created, and existing columns are extended with `nil` values.

##### `AppendRowsByColName(rowsData ...map[string]interface{}) *DataTable`
Appends rows to the `DataTable` by mapping column names to values. New columns are created if necessary, and existing columns are extended with `nil` values.

##### `GetElement(rowIndex int, columnIndex string) interface{}`
Returns the element at the specified row and column index. If the indices are out of bounds, it returns `nil`.

##### `GetElementByNumberIndex(rowIndex int, columnIndex int) interface{}`
Returns the element at the specified row and column number index. If the indices are out of bounds, it returns `nil`.

##### `GetCol(index string) *DataList`
Returns a `DataList` containing the data of the specified column index.

##### `GetColByNumber(index int) *DataList`
Returns a `DataList` containing the data of the specified column number.

##### `GetRow(index int) *DataList`
Returns a `DataList` containing the data of the specified row index.

##### `UpdateElement(rowIndex int, columnIndex string, value interface{})`
Updates the element at the specified row and column index.

##### `UpdateCol(index string, dl *DataList)`
Updates the column with the specified index with a new `DataList`.

##### `UpdateColByNumber(index int, dl *DataList)`
Updates the column at the specified number index with a new `DataList`.

##### `UpdateRow(index int, dl *DataList)`
Updates the row at the specified index with a new `DataList`.

##### `SetColToRowNames(columnIndex string) *DataTable`
Sets the row names to the values of the specified column and drops the column.

##### `SetRowToColNames(rowIndex int) *DataTable`
Sets the column names to the values of the specified row and drops the row.

##### `FindRowsIfContains(value interface{}) []int`
Returns the indices of rows that contain the specified value.

##### `FindRowsIfContainsAll(values ...interface{}) []int`
Returns the indices of rows that contain all specified values.

##### `FindRowsIfAnyElementContainsSubstring(substring string) []int`
Returns the indices of rows that contain at least one element with the specified substring.

##### `FindRowsIfAllElementsContainSubstring(substring string) []int`
Returns the indices of rows where all elements contain the specified substring.

##### `FindColsIfContains(value interface{}) []string`
Returns the names of columns that contain the specified value.

##### `FindColsIfContainsAll(values ...interface{}) []string`
Returns the names of columns that contain all specified values.

##### `FindColsIfAnyElementContainsSubstring(substring string) []string`
Returns the names of columns that contain at least one element with the specified substring.

##### `FindColsIfAllElementsContainSubstring(substring string) []string`
Returns the names of columns where all elements contain the specified substring.

##### `DropColsByName(columnNames ...string)`
Drops columns from the `DataTable` by their names.

##### `DropColsByIndex(columnIndices ...string)`
Drops columns from the `DataTable` by their indices.

##### `DropColsByNumber(columnIndices ...int)`
Drops columns from the `DataTable` by their number indices.

##### `DropColsContainStringElements()`
Drops columns that contain any string elements.

##### `DropColsContainNumbers()`
Drops columns that contain any numeric elements.

##### `DropColsContainNil()`
Drops columns that contain any `nil` elements.

##### `DropRowsByIndex(rowIndices ...int)`
Drops rows from the `DataTable` by their indices.

##### `DropRowsByName(rowNames ...string)`
Drops rows from the `DataTable` by their names.

##### `DropRowsContainStringElements()`
Drops rows that contain any string elements.

##### `DropRowsContainNumbers()`
Drops rows that contain any numeric elements.

##### `DropRowsContainNil()`
Drops rows that contain any `nil` elements.

##### `Data(useNamesAsKeys ...bool) map[string][]interface{}`
Returns a map representing the data in the `DataTable`. The keys can be column names if specified.

##### `Show()`
Prints the content of the `DataTable` in a tabular format.

##### `ShowTypes()`
Prints the types of elements in the `DataTable`.

##### `Size() (int, int)`
Returns the number of rows and columns in the `DataTable`.
The first return value is the number of rows, and the second return value is the number of columns.

##### `Count(value interface{}) int`
Returns the number of occurrences of the specified value in the `DataTable`.

##### `Counter() map[interface{}]int`
Returns a map of the number of occurrences of each value in the `DataTable`.

##### `Transpose() *DataTable`
Transposes the `DataTable`, converting rows into columns and vice versa.

##### `Mean() interface{}`
Returns the mean of the `DataTable`.

##### `GetRowNameByIndex(index int) string`
Returns the name of the row at the specified index.

##### `SetRowNameByIndex(index int, name string)`
Sets the name of the row at the specified index.

##### `GetCreationTimestamp() int64`
Returns the timestamp when the `DataTable` was created.

##### `GetLastModifiedTimestamp() int64`
Returns the timestamp when the `DataTable` was last modified.

##### `FilterFilter(filterFunc FilterFunc) *DataTable`
Returns a new `DataTable` with rows that satisfy the given filter function.

##### `FilterByCustomElement(f func(value interface{}) bool) *DataTable`
Returns a new `DataTable` with rows where any element satisfies the custom filter function.

##### `FilterByColIndexGreaterThan(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is greater than the specified threshold.

##### `FilterByColIndexGreaterThanOrEqualTo(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is greater than or equal to the specified threshold.

##### `FilterByColIndexLessThan(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is less than the specified threshold.

##### `FilterByColIndexLessThanOrEqualTo(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is less than or equal to the specified threshold.

##### `FilterByColIndexEqualTo(index string) *DataTable`
Returns a new `DataTable` with rows where the column index equals the specified index.

##### `FilterByColNameEqualTo(name string) *DataTable`
Returns a new `DataTable` with rows where the column name equals the specified name.

##### `FilterByColNameContains(substring string) *DataTable`
Returns a new `DataTable` with rows where the column name contains the specified substring.

##### `FilterByRowNameEqualTo(name string) *DataTable`
Returns a new `DataTable` with rows where the row name equals the specified name.

##### `FilterByRowNameContains(substring string) *DataTable`
Returns a new `DataTable` with rows where the row name contains the specified substring.

##### `FilterByRowIndexGreaterThan(threshold int) *DataTable`
Returns a new `DataTable` with rows where the row index is greater than the specified threshold.

##### `FilterByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable`
Returns a new `DataTable` with rows where the row index is greater than or equal to the specified threshold.

##### `FilterByRowIndexLessThan(threshold int) *DataTable`
Returns a new `DataTable` with rows where the row index is less than the specified threshold.

##### `FilterByRowIndexLessThanOrEqualTo(threshold int) *DataTable`
Returns a new `DataTable` with rows where the row index is less than or equal to the specified threshold.

##### `FilterByRowIndexEqualTo(index int) *DataTable`
Returns a new `DataTable` with rows where the row index equals the specified index.

#### `ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error`
Writes the `DataTable` to a CSV file. If `setRowNamesToFirstCol` is `true`, the first column will be used as row names. If `setColNamesToFirstRow` is `true`, the first row will be used as column names. If `includeBOM` is `true`, a BOM will be included at the beginning of the file(compatible with excel).

#### `LoadFromCSV(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool) error`
Loads a `DataTable` from a CSV file. If `setFirstColToRowNames` is `true`, the first column will be used as row names. If `setFirstRowToColNames` is `true`, the first row will be used as column names.

#### `ToJSON(filePath string, useColNames bool) error`
Writes the `DataTable` to a JSON file.

#### `ToJSON_Bytes(useColNames bool) []byte`
Returns the JSON representation of the `DataTable` as a byte array.

## Best Practices

1. Use appropriate method calls to manipulate data (e.g., use AppendRowsByName when you have named columns).
2. Regularly check for and handle nil values in your data.
3. Use the Show() method for debugging and data inspection.
4. Monitor logs for warnings and errors during operations.
5. Consider the trade-offs between using named rows/columns and index-based access based on your use case.
6. When performing multiple operations, consider grouping them to minimize the number of times the lastModifiedTimestamp is updated.
