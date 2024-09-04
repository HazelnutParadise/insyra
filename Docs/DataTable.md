# DataTable

## Overview

DataTable is a flexible and efficient data structure for managing two-dimensional data in Go. It provides a variety of methods for data manipulation, querying, and display. The DataTable structure uses a column-based storage model, which allows for efficient column operations and flexible row management.

## Structure

```go
type DataTable struct {
    mu                    sync.Mutex
    columns               []*DataList
    columnIndex           map[string]int
    rowNames              map[string]int
    creationTimestamp     int64
    lastModifiedTimestamp int64
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

##### `AppendColumns(columns ...*DataList)`
Appends columns to the `DataTable`. If the new columns are shorter than existing ones, `nil` values will be appended to match the length. If they are longer, existing columns will be extended with `nil` values.

##### `AppendRowsFromDataList(rowsData ...*DataList)`
Appends rows to the `DataTable` using `DataList` objects. If the new rows are shorter than existing columns, `nil` values will be appended to match the length.

##### `AppendRowsByIndex(rowsData ...map[string]interface{})`
Appends rows to the `DataTable` by mapping column indices to values. If necessary, new columns are created, and existing columns are extended with `nil` values.

##### `AppendRowsByName(rowsData ...map[string]interface{})`
Appends rows to the `DataTable` by mapping column names to values. New columns are created if necessary, and existing columns are extended with `nil` values.

##### `GetElement(rowIndex int, columnIndex string) interface{}`
Returns the element at the specified row and column index. If the indices are out of bounds, it returns `nil`.

##### `GetColumn(index string) *DataList`
Returns a `DataList` containing the data of the specified column index.

##### `GetRow(index int) *DataList`
Returns a `DataList` containing the data of the specified row index.

##### `UpdateElement(rowIndex int, columnIndex string, value interface{})`
Updates the element at the specified row and column index.

##### `UpdateColumn(index string, dl *DataList)`
Updates the column with the specified index with a new `DataList`.

##### `UpdateRow(index int, dl *DataList)`
Updates the row at the specified index with a new `DataList`.

##### `FindRowsIfContains(value interface{}) []int`
Returns the indices of rows that contain the specified value.

##### `FindRowsIfContainsAll(values ...interface{}) []int`
Returns the indices of rows that contain all specified values.

##### `FindRowsIfAnyElementContainsSubstring(substring string) []int`
Returns the indices of rows that contain at least one element with the specified substring.

##### `FindRowsIfAllElementsContainSubstring(substring string) []int`
Returns the indices of rows where all elements contain the specified substring.

##### `FindColumnsIfContains(value interface{}) []string`
Returns the names of columns that contain the specified value.

##### `FindColumnsIfContainsAll(values ...interface{}) []string`
Returns the names of columns that contain all specified values.

##### `FindColumnsIfAnyElementContainsSubstring(substring string) []string`
Returns the names of columns that contain at least one element with the specified substring.

##### `FindColumnsIfAllElementsContainSubstring(substring string) []string`
Returns the names of columns where all elements contain the specified substring.

##### `DropColumnsByName(columnNames ...string)`
Drops columns from the `DataTable` by their names.

##### `DropColumnsByIndex(columnIndices ...string)`
Drops columns from the `DataTable` by their indices.

##### `DropColumnsContainStringElements()`
Drops columns that contain any string elements.

##### `DropColumnsContainNumbers()`
Drops columns that contain any numeric elements.

##### `DropColumnsContainNil()`
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

##### `FilterByColumnIndexGreaterThan(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is greater than the specified threshold.

##### `FilterByColumnIndexGreaterThanOrEqualTo(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is greater than or equal to the specified threshold.

##### `FilterByColumnIndexLessThan(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is less than the specified threshold.

##### `FilterByColumnIndexLessThanOrEqualTo(threshold string) *DataTable`
Returns a new `DataTable` with rows where the column index is less than or equal to the specified threshold.

##### `FilterByColumnIndexEqualTo(index string) *DataTable`
Returns a new `DataTable` with rows where the column index equals the specified index.

##### `FilterByColumnNameEqualTo(name string) *DataTable`
Returns a new `DataTable` with rows where the column name equals the specified name.

##### `FilterByColumnNameContains(substring string) *DataTable`
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

#### `ToCSV(filePath string, setRowNamesToFirstColumn bool, setColumnNamesToFirstRow bool) error`
Writes the `DataTable` to a CSV file. If `setRowNamesToFirstColumn` is `true`, the first column will be used as row names. If `setColumnNamesToFirstRow` is `true`, the first row will be used as column names.

#### `LoadFromCSV(filePath string, setFirstColumnToRowNames bool, setFirstRowToColumnNames bool) error`
Loads a `DataTable` from a CSV file. If `setFirstColumnToRowNames` is `true`, the first column will be used as row names. If `setFirstRowToColumnNames` is `true`, the first row will be used as column names.

## Best Practices

1. Use appropriate method calls to manipulate data (e.g., use AppendRowsByName when you have named columns).
2. Regularly check for and handle nil values in your data.
3. Use the Show() method for debugging and data inspection.
4. Monitor logs for warnings and errors during operations.
5. Consider the trade-offs between using named rows/columns and index-based access based on your use case.
6. When performing multiple operations, consider grouping them to minimize the number of times the lastModifiedTimestamp is updated.
