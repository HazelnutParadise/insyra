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

### NewDataTable

```go
func NewDataTable(columns ...*DataList) *DataTable
```

Creates a new DataTable with optional initial columns.

## Methods

### Data Manipulation

#### AppendColumns

```go
func (dt *DataTable) AppendColumns(columns ...*DataList)
```

Appends new columns to the DataTable.

#### AppendRowsFromDataList

```go
func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList)
```

Appends new rows to the DataTable using DataList objects.

#### AppendRowsByIndex

```go
func (dt *DataTable) AppendRowsByIndex(rowsData ...map[string]interface{})
```

Appends new rows to the DataTable using maps with column indices as keys.

#### AppendRowsByName

```go
func (dt *DataTable) AppendRowsByName(rowsData ...map[string]interface{})
```

Appends new rows to the DataTable using maps with column names as keys.

### Data Retrieval

#### GetElement

```go
func (dt *DataTable) GetElement(rowIndex int, columnIndex string) interface{}
```

Retrieves an element at the specified row and column.

#### GetColumn

```go
func (dt *DataTable) GetColumn(index string) *DataList
```

Retrieves a column by its index.

#### GetRow

```go
func (dt *DataTable) GetRow(index int) *DataList
```

Retrieves a row by its index.

### Data Update

#### UpdateElement

```go
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value interface{})
```

Updates an element at the specified row and column.

#### UpdateColumn

```go
func (dt *DataTable) UpdateColumn(index string, dl *DataList)
```

Updates an entire column.

#### UpdateRow

```go
func (dt *DataTable) UpdateRow(index int, dl *DataList)
```

Updates an entire row.

### Data Query

#### FindRowsIfContains

```go
func (dt *DataTable) FindRowsIfContains(value interface{}) []int
```

Finds rows that contain the specified value.

#### FindRowsIfContainsAll

```go
func (dt *DataTable) FindRowsIfContainsAll(values ...interface{}) []int
```

Finds rows that contain all specified values.

#### FindRowsIfAnyElementContainsSubstring

```go
func (dt *DataTable) FindRowsIfAnyElementContainsSubstring(substring string) []int
```

Finds rows where any element contains the specified substring.

#### FindRowsIfAllElementsContainSubstring

```go
func (dt *DataTable) FindRowsIfAllElementsContainSubstring(substring string) []int
```

Finds rows where all elements contain the specified substring.

#### FindColumnsIfContains

```go
func (dt *DataTable) FindColumnsIfContains(value interface{}) []string
```

Finds columns that contain the specified value.

#### FindColumnsIfContainsAll

```go
func (dt *DataTable) FindColumnsIfContainsAll(values ...interface{}) []string
```

Finds columns that contain all specified values.

#### FindColumnsIfAnyElementContainsSubstring

```go
func (dt *DataTable) FindColumnsIfAnyElementContainsSubstring(substring string) []string
```

Finds columns where any element contains the specified substring.

#### FindColumnsIfAllElementsContainSubstring

```go
func (dt *DataTable) FindColumnsIfAllElementsContainSubstring(substring string) []string
```

Finds columns where all elements contain the specified substring.

### Data Deletion

#### DropColumnsByName

```go
func (dt *DataTable) DropColumnsByName(columnNames ...string)
```

Drops columns by their names.

#### DropColumnsByIndex

```go
func (dt *DataTable) DropColumnsByIndex(columnIndices ...string)
```

Drops columns by their indices.

#### DropColumnsContainStringElements

```go
func (dt *DataTable) DropColumnsContainStringElements()
```

Drops columns that contain string elements.

#### DropColumnsContainNumbers

```go
func (dt *DataTable) DropColumnsContainNumbers()
```

Drops columns that contain number elements.

#### DropColumnsContainNil

```go
func (dt *DataTable) DropColumnsContainNil()
```

Drops columns that contain nil elements.

#### DropRowsByIndex

```go
func (dt *DataTable) DropRowsByIndex(rowIndices ...int)
```

Drops rows by their indices.

#### DropRowsByName

```go
func (dt *DataTable) DropRowsByName(rowNames ...string)
```

Drops rows by their names.

#### DropRowsContainStringElements

```go
func (dt *DataTable) DropRowsContainStringElements()
```

Drops rows that contain string elements.

#### DropRowsContainNumbers

```go
func (dt *DataTable) DropRowsContainNumbers()
```

Drops rows that contain number elements.

#### DropRowsContainNil

```go
func (dt *DataTable) DropRowsContainNil()
```

Drops rows that contain nil elements.

### Utility Methods

#### Data

```go
func (dt *DataTable) Data(useNamesAsKeys ...bool) map[string][]interface{}
```

Returns the DataTable's data as a map.

#### Show

```go
func (dt *DataTable) Show()
```

Displays the DataTable in a formatted manner.

#### GetRowNameByIndex

```go
func (dt *DataTable) GetRowNameByIndex(index int) string
```

Retrieves the name of a row by its index.

#### SetRowNameByIndex

```go
func (dt *DataTable) SetRowNameByIndex(index int, name string)
```

Sets the name of a row by its index.

#### GetCreationTimestamp

```go
func (dt *DataTable) GetCreationTimestamp() int64
```

Retrieves the creation timestamp of the DataTable.

#### GetLastModifiedTimestamp

```go
func (dt *DataTable) GetLastModifiedTimestamp() int64
```

Retrieves the last modified timestamp of the DataTable.

### Internal Utility Methods

These methods are used internally but are part of the public interface:

#### getSortedColumnNames

```go
func (dt *DataTable) getSortedColumnNames() []string
```

Returns a sorted list of column names.

#### getRowNameByIndex

```go
func (dt *DataTable) getRowNameByIndex(index int) (string, bool)
```

Retrieves the name of a row by its index, returning a bool indicating if the name exists.

#### getMaxColumnLength

```go
func (dt *DataTable) getMaxColumnLength() int
```

Returns the maximum length among all columns.

#### updateTimestamp

```go
func (dt *DataTable) updateTimestamp()
```

Updates the last modified timestamp of the DataTable.

## Thread Safety

All methods in DataTable are thread-safe, using a mutex to ensure safe concurrent access.

## Error Handling

The DataTable uses a logging system to report warnings and errors. Users should monitor these logs for potential issues during operations.

## Best Practices

1. Use appropriate method calls to manipulate data (e.g., use AppendRowsByName when you have named columns).
2. Regularly check for and handle nil values in your data.
3. Use the Show() method for debugging and data inspection.
4. Monitor logs for warnings and errors during operations.
5. Consider the trade-offs between using named rows/columns and index-based access based on your use case.
6. When performing multiple operations, consider grouping them to minimize the number of times the lastModifiedTimestamp is updated.
