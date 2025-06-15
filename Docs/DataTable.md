# DataTable

DataTable is the core data structure of Insyra for handling structured data. It provides rich data manipulation functionality including reading, writing, filtering, statistical analysis, and transformation operations.

## Table of Contents

- [Data Structure](#data-structure)
- [Creating DataTable](#creating-datatable)
- [Data Loading](#data-loading)
- [Data Saving](#data-saving)
- [Data Operations](#data-operations)
- [Column Calculation](#column-calculation)
- [Searching](#searching)
- [Filtering](#filtering)
- [Statistical Analysis](#statistical-analysis)
- [Utility Methods](#utility-methods)
- [Notes](#notes)

## Data Structure

### DataTable Struct

```go
type DataTable struct {
    columns               []*DataList            // Data columns
    columnIndex           map[string]int         // Column index mapping
    rowNames              map[string]int         // Row name mapping
    name                  string                 // Table name
    creationTimestamp     int64                  // Creation timestamp
    lastModifiedTimestamp atomic.Int64           // Last modified timestamp
}
```

**Field Descriptions:**

- `columns`: Slice of DataList pointers representing table columns
- `columnIndex`: Maps column indices (A, B, C...) to slice positions
- `rowNames`: Maps row names to their indices
- `name`: Name of the DataTable
- `creationTimestamp`: Unix timestamp when the table was created
- `lastModifiedTimestamp`: Unix timestamp when the table was last modified

## Creating DataTable

### NewDataTable

Creates a new empty DataTable.

```go
func NewDataTable(columns ...*DataList) *DataTable
```

**Parameters:**

- `columns`: Optional DataList columns to initialize the table with

**Returns:**

- `*DataTable`: A newly created DataTable

**Example:**

```go
dt := insyra.NewDataTable()

// Or create with initial columns
col1 := insyra.NewDataList("Alice", "Bob", "Charlie")
col2 := insyra.NewDataList(25, 30, 35)
dt := insyra.NewDataTable(col1, col2)
```

## Data Loading

### LoadFromCSV

Loads data from a CSV file.

```go
func (dt *DataTable) LoadFromCSV(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool) error
```

**Parameters:**

- `filePath`: CSV file path
- `setFirstColToRowNames`: Whether to use the first column as row names
- `setFirstRowToColNames`: Whether to use the first row as column names

**Returns:**

- `error`: Error information, returns nil if successful

**Example:**

```go
dt := insyra.NewDataTable()
err := dt.LoadFromCSV("data.csv", false, true)
if err != nil {
    log.Fatal(err)
}
```

### LoadFromJSON

Loads data from a JSON file.

```go
func (dt *DataTable) LoadFromJSON(filePath string) error
```

**Parameters:**

- `filePath`: JSON file path

**Returns:**

- `error`: Error information, returns nil if successful

**Example:**

```go
dt := insyra.NewDataTable()
err := dt.LoadFromJSON("data.json")
if err != nil {
    log.Fatal(err)
}
```

### LoadFromJSON_Bytes

Loads data from JSON byte data.

```go
func (dt *DataTable) LoadFromJSON_Bytes(data []byte) error
```

**Parameters:**

- `data`: JSON data as byte slice

**Returns:**

- `error`: Error information, returns nil if successful

**Example:**

```go
dt := insyra.NewDataTable()
jsonData := []byte(`[{"name":"John","age":30},{"name":"Jane","age":25}]`)
err := dt.LoadFromJSON_Bytes(jsonData)
if err != nil {
    log.Fatal(err)
}
```

### ReadSQL

Loads data from SQL query results.

```go
func ReadSQL(db *sql.DB, query string) (*DataTable, error)
```

**Parameters:**

- `db`: Database connection
- `query`: SQL query statement

**Returns:**

- `*DataTable`: DataTable loaded with data
- `error`: Error information, returns nil if successful

**Example:**

```go
db, _ := sql.Open("mysql", "connection_string")
dt, err := insyra.ReadSQL(db, "SELECT * FROM users")
if err != nil {
    log.Fatal(err)
}
```

## Data Saving

### ToCSV

Saves the DataTable as a CSV file.

```go
func (dt *DataTable) ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error
```

**Parameters:**

- `filePath`: Output CSV file path
- `setRowNamesToFirstCol`: Whether to include row names as the first column
- `setColNamesToFirstRow`: Whether to include column names as the first row
- `includeBOM`: Whether to include BOM (Byte Order Mark) in the file

**Returns:**

- `error`: Error information, returns nil if successful

**Example:**

```go
err := dt.ToCSV("output.csv", false, true, false)
if err != nil {
    log.Fatal(err)
}
```

### ToJSON

Saves the DataTable as a JSON file.

```go
func (dt *DataTable) ToJSON(filePath string, useColNames bool) error
```

**Parameters:**

- `filePath`: Output JSON file path
- `useColNames`: Whether to use column names as keys in JSON objects

**Returns:**

- `error`: Error information, returns nil if successful

**Example:**

```go
err := dt.ToJSON("output.json", true)
if err != nil {
    log.Fatal(err)
}
```

### ToJSON_Bytes

Converts the DataTable to JSON format and returns as bytes.

```go
func (dt *DataTable) ToJSON_Bytes(useColNames bool) []byte
```

**Parameters:**

- `useColNames`: Whether to use column names as keys in JSON objects

**Returns:**

- `[]byte`: JSON data as byte slice

**Example:**

```go
jsonData := dt.ToJSON_Bytes(true)
fmt.Println(string(jsonData))
```

### ToSQL

Writes the DataTable to a SQL database.

```go
func (dt *DataTable) ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error
```

**Parameters:**

- `db`: GORM database connection
- `tableName`: Target table name
- `options`: Optional SQL options

**Returns:**

- `error`: Error information, returns nil if successful

**Example:**

```go
db, _ := gorm.Open(mysql.Open("connection_string"))
err := dt.ToSQL(db, "users")
if err != nil {
    log.Fatal(err)
}
```

## Data Operations

### AppendCols

Appends columns to the DataTable.

```go
func (dt *DataTable) AppendCols(columns ...*DataList) *DataTable
```

**Parameters:**

- `columns`: DataList columns to append

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
col := insyra.NewDataList(1, 2, 3, 4)
dt.AppendCols(col)
```

### AppendRowsByColIndex

Appends rows using column indices.

```go
func (dt *DataTable) AppendRowsByColIndex(rowsData ...map[string]any) *DataTable
```

**Parameters:**

- `rowsData`: Row data maps where keys are column indices (A, B, C...)

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.AppendRowsByColIndex(map[string]any{
    "A": "John",
    "B": 25,
    "C": "Engineer",
})
```

### AppendRowsByColName

Appends rows using column names.

```go
func (dt *DataTable) AppendRowsByColName(rowsData ...map[string]any) *DataTable
```

**Parameters:**

- `rowsData`: Row data maps where keys are column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.AppendRowsByColName(map[string]any{
    "name": "John",
    "age":  25,
    "role": "Engineer",
})
```

### AppendRowsFromDataList

Appends rows to the DataTable, with each row represented by a DataList.

```go
func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) *DataTable
```

**Parameters:**

- `rowsData`: DataList objects representing rows to append

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
row := insyra.NewDataList("John", 25, "Engineer")
dt.AppendRowsFromDataList(row)
```

### GetElement

Gets the value at a specific row and column.

```go
func (dt *DataTable) GetElement(rowIndex int, columnIndex string) any
```

**Parameters:**

- `rowIndex`: Row index (0-based)
- `columnIndex`: Column index (A, B, C...)

**Returns:**

- `any`: Value at the specified position

**Example:**

```go
value := dt.GetElement(0, "A")
fmt.Println(value)
```

### GetCol

Gets a column by its index.

```go
func (dt *DataTable) GetCol(index string) *DataList
```

**Parameters:**

- `index`: Column index (A, B, C...)

**Returns:**

- `*DataList`: The specified column

**Example:**

```go
column := dt.GetCol("A")
```

### GetColByNumber

Gets a column by its numeric index.

```go
func (dt *DataTable) GetColByNumber(index int) *DataList
```

**Parameters:**

- `index`: Numeric column index (0-based)

**Returns:**

- `*DataList`: The specified column

**Example:**

```go
column := dt.GetColByNumber(0)
```

### GetColByName

Gets a column by its name.

```go
func (dt *DataTable) GetColByName(name string) *DataList
```

**Parameters:**

- `name`: The name of the column.

**Returns:**

- `*DataList`: The specified column.

**Example:**

```go
column := dt.GetColByName("column_name")
```

### GetRow

Gets a row by its index.

```go
func (dt *DataTable) GetRow(index int) *DataList
```

**Parameters:**

- `index`: Row index (0-based)

**Returns:**

- `*DataList`: The specified row

**Example:**

```go
row := dt.GetRow(0)
```

### GetRowByName

Gets a row by its name.

```go
func (dt *DataTable) GetRowByName(name string) *DataList
```

**Parameters:**

- `name`: The name of the row.

**Returns:**

- `*DataList`: The specified row. Returns nil if the row name is not found.

**Example:**

```go
row := dt.GetRowByName("row_name")
```

### UpdateElement

Updates the value at a specific position.

```go
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value any)
```

**Parameters:**

- `rowIndex`: Row index (0-based)
- `columnIndex`: Column index (A, B, C...)
- `value`: New value

**Example:**

```go
dt.UpdateElement(0, "A", "Jane")
```

### UpdateCol

Updates an entire column with new data.

```go
func (dt *DataTable) UpdateCol(index string, dl *DataList)
```

**Parameters:**

- `index`: Column index (A, B, C...)
- `dl`: New DataList to replace the column

**Example:**

```go
newCol := insyra.NewDataList(1, 2, 3, 4)
dt.UpdateCol("A", newCol)
```

### UpdateColByNumber

Updates an entire column by its numeric index.

```go
func (dt *DataTable) UpdateColByNumber(index int, dl *DataList)
```

**Parameters:**

- `index`: Numeric column index (0-based)
- `dl`: New DataList to replace the column

**Example:**

```go
newCol := insyra.NewDataList(1, 2, 3, 4)
dt.UpdateColByNumber(0, newCol)
```

### UpdateRow

Updates an entire row with new data.

```go
func (dt *DataTable) UpdateRow(index int, dl *DataList)
```

**Parameters:**

- `index`: Row index (0-based)
- `dl`: New DataList to replace the row

**Example:**

```go
newRow := insyra.NewDataList("Jane", 28, "Manager")
dt.UpdateRow(0, newRow)
```

### GetElementByNumberIndex

Gets the value at a specific row and column using numeric indices.

```go
func (dt *DataTable) GetElementByNumberIndex(rowIndex int, columnIndex int) any
```

**Parameters:**

- `rowIndex`: Row index (0-based)
- `columnIndex`: Numeric column index (0-based)

**Returns:**

- `any`: Value at the specified position

**Example:**

```go
value := dt.GetElementByNumberIndex(0, 0)
fmt.Println(value)
```

### SetColToRowNames

Sets the row names to the values of the specified column and drops the column.

```go
func (dt *DataTable) SetColToRowNames(columnIndex string) *DataTable
```

**Parameters:**

- `columnIndex`: Column index (A, B, C...) to use as row names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetColToRowNames("A") // Use column A values as row names
```

### SetRowToColNames

Sets the column names to the values of the specified row and drops the row.

```go
func (dt *DataTable) SetRowToColNames(rowIndex int) *DataTable
```

**Parameters:**

- `rowIndex`: Row index (0-based) to use as column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetRowToColNames(0) // Use first row values as column names
```

### ChangeRowName

Changes the name of a row.

```go
func (dt *DataTable) ChangeRowName(oldName, newName string) *DataTable
```

**Parameters:**

- `oldName`: The current name of the row.
- `newName`: The new name for the row.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.ChangeRowName("old_row_name", "new_row_name")
```

### SetRowNameByIndex

Sets the name of a row by its index.

```go
func (dt *DataTable) SetRowNameByIndex(index int, name string)
```

**Parameters:**

- `index`: Row index (0-based)
- `name`: New row name

**Example:**

```go
dt.SetRowNameByIndex(0, "FirstRow") // Set the name of the first row to "FirstRow"
```

### GetRowNameByIndex

Gets the name of a row by its index.

```go
func (dt *DataTable) GetRowNameByIndex(index int) string
```

**Parameters:**

- `index`: The numeric index of the row (0-based).

**Returns:**

- `string`: The name of the row. Returns an empty string if the row does not exist or has no name.

**Example:**

```go
rowName := dt.GetRowNameByIndex(0)
fmt.Printf("Name of the first row: %s\\n", rowName)
```

### ChangeColName

Changes the name of a column.

```go
func (dt *DataTable) ChangeColName(oldName, newName string) *DataTable
```

**Parameters:**

- `oldName`: The current name of the column.
- `newName`: The new name for the column.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.ChangeColName("old_column_name", "new_column_name")
```

### GetColNameByNumber

Gets the name of a column by its numeric index.

```go
func (dt *DataTable) GetColNameByNumber(index int) string
```

**Parameters:**

- `index`: The numeric index of the column (0-based).

**Returns:**

- `string`: The name of the column.

**Example:**

```go
columnName := dt.GetColNameByNumber(0)
fmt.Printf("Name of the first column: %s\\n", columnName)
```

### SetColNameByNumber

Sets the name of a column by its numeric index.

```go
func (dt *DataTable) SetColNameByNumber(numberIndex int, name string) *DataTable
```

**Parameters:**

- `numberIndex`: Numeric column index (0-based)
- `name`: New column name

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetColNameByNumber(0, "ID") // Set the name of the first column to "ID"
```

### SetColNameByIndex

Sets the name of a column by its alphabetical index.

```go
func (dt *DataTable) SetColNameByIndex(index string, name string) *DataTable
```

**Parameters:**

- `index`: Alphabetical column index (A, B, C...)
- `name`: New column name

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetColNameByIndex("A", "Identifier") // Set the name of column A to "Identifier"
```

### ColNamesToFirstRow

Moves all column names to the first row of data and keeps the names as column headers.

```go
func (dt *DataTable) ColNamesToFirstRow() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// If DataTable has columns named "Name", "Age", "Role"
// After calling ColNamesToFirstRow(), the first row will contain ["Name", "Age", "Role"]
dt.ColNamesToFirstRow()
```

### DropColNames

Removes all column names, setting them to empty strings.

```go
func (dt *DataTable) DropColNames() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// Remove all column names
dt.DropColNames()

// Can be chained with other methods
dt.ColNamesToFirstRow().DropColNames().Show()
```

### DropRowsByName

Drops rows by their names.

```go
func (dt *DataTable) DropRowsByName(rowNames ...string) *DataTable
```

**Parameters:**

- `rowNames`: Names of rows to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsByName("row1", "row2")
```

### DropColsByIndex

Drops columns by their indices.

```go
func (dt *DataTable) DropColsByIndex(columnIndices ...string)
```

**Parameters:**

- `columnIndices`: Column indices (A, B, C...) to drop

**Example:**

```go
dt.DropColsByIndex("A", "C") // Drops columns A and C
```

### DropColsByNumber

Drops columns by their numeric indices.

```go
func (dt *DataTable) DropColsByNumber(columnIndices ...int)
```

**Parameters:**

- `columnIndices`: Numeric column indices (0-based) to drop

**Example:**

```go
dt.DropColsByNumber(0, 2) // Drops first and third columns
```

### DropColsContainStringElements

Drops columns that contain string elements.

```go
func (dt *DataTable) DropColsContainStringElements() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainStringElements()
```

### DropColsContainNumbers

Drops columns that contain numeric elements.

```go
func (dt *DataTable) DropColsContainNumbers() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNumbers()
```

### DropColsContainNil

Drops columns that contain nil elements.

```go
func (dt *DataTable) DropColsContainNil() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNil()
```

### DropRowsContainStringElements

Drops rows that contain string elements.

```go
func (dt *DataTable) DropRowsContainStringElements() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainStringElements()
```

### DropRowsContainNumbers

Drops rows that contain numeric elements.

```go
func (dt *DataTable) DropRowsContainNumbers() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNumbers()
```

### DropRowsContainNil

Drops rows that contain nil elements.

```go
func (dt *DataTable) DropRowsContainNil() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNil()
```

### SwapColsByName

Swaps two columns by their names.

```go
func (dt *DataTable) SwapColsByName(columnName1 string, columnName2 string) *DataTable
```

**Parameters:**

- `columnName1`: Name of the first column to swap.
- `columnName2`: Name of the second column to swap.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.SwapColsByName("ColumnA", "ColumnB")
```

### SwapColsByIndex

Swaps two columns by their letter indices (e.g., "A", "B").

```go
func (dt *DataTable) SwapColsByIndex(columnIndex1 string, columnIndex2 string) *DataTable
```

**Parameters:**

- `columnIndex1`: Letter index of the first column.
- `columnIndex2`: Letter index of the second column.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.SwapColsByIndex("A", "C")
```

### SwapColsByNumber

Swaps two columns by their numerical indices (0-based).

```go
func (dt *DataTable) SwapColsByNumber(columnNumber1 int, columnNumber2 int) *DataTable
```

**Parameters:**

- `columnNumber1`: Numerical index of the first column.
- `columnNumber2`: Numerical index of the second column.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.SwapColsByNumber(0, 2)
```

### SwapRowsByIndex

Swaps two rows by their numerical indices (0-based).

```go
func (dt *DataTable) SwapRowsByIndex(rowIndex1 int, rowIndex2 int) *DataTable
```

**Parameters:**

- `rowIndex1`: Numerical index of the first row.
- `rowIndex2`: Numerical index of the second row.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.SwapRowsByIndex(0, 1)
```

### SwapRowsByName

Swaps two rows by their names.

```go
func (dt *DataTable) SwapRowsByName(rowName1 string, rowName2 string) *DataTable
```

**Parameters:**

- `rowName1`: Name of the first row to swap.
- `rowName2`: Name of the second row to swap.

**Returns:**

- `*DataTable`: The modified DataTable.

**Example:**

```go
dt.SwapRowsByName("RowX", "RowY")
```

### Count

Returns the number of rows in the DataTable.

```go
func (dt *DataTable) Count() int
```

**Returns:**

- `int`: Number of rows

**Example:**

```go
count := dt.Count()
fmt.Printf("Table has %d rows\n", count)
```

### Counter

Returns the number of occurrences of each value in the DataTable.

```go
func (dt *DataTable) Counter() map[any]int
```

**Returns:**

- `map[any]int`: Map containing the count of each value in the DataTable

**Example:**

```go
counts := dt.Counter()
fmt.Printf("Value counts: %v\n", counts)
```

### GetCreationTimestamp

Gets the creation timestamp of the DataTable.

```go
func (dt *DataTable) GetCreationTimestamp() int64
```

**Returns:**

- `int64`: Unix timestamp when the table was created

**Example:**

```go
timestamp := dt.GetCreationTimestamp()
fmt.Printf("Created at: %d\n", timestamp)
```

### GetLastModifiedTimestamp

Gets the last modified timestamp of the DataTable.

```go
func (dt *DataTable) GetLastModifiedTimestamp() int64
```

**Returns:**

- `int64`: Unix timestamp when the table was last modified

**Example:**

```go
timestamp := dt.GetLastModifiedTimestamp()
fmt.Printf("Last modified at: %d\n", timestamp)
```

## Column Calculation

### AddColUsingCCL

Adds a new column to the DataTable by evaluating a Column Calculation Language (CCL) expression on each row.

```go
func (dt *DataTable) AddColUsingCCL(newColName, ccl string) *DataTable
```

**Parameters:**

- `newColName`: Name of the new column to be created
- `ccl`: CCL expression to evaluate

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Adding a column that contains conditional values
dt.AddColUsingCCL("age_group", "IF(B > 30, 'Senior', 'Junior')")

// Using math operations
dt.AddColUsingCCL("total", "A + B + C")

// Using logical operators
dt.AddColUsingCCL("is_valid", "AND(A > 0, B < 100)")

// Nested conditions
dt.AddColUsingCCL("category", "IF(A > 90, 'Excellent', IF(A > 70, 'Good', IF(A > 50, 'Average', 'Poor')))")

// Using CASE for multiple conditions
dt.AddColUsingCCL("status", "CASE(A > 90, 'A', A > 80, 'B', A > 70, 'C', 'F')")

// Using range comparisons
dt.AddColUsingCCL("in_range", "IF(10 < A <= 20, 'In range', 'Out of range')")
```

**Notes:**

- CCL allows performing calculations and operations on columns in a DataTable
- Column references use Excel-style notation (A, B, C...) for simplicity
- Supports mathematical operations, logical operations, conditionals and more
- See [CCL Documentation](CCL.md) for a comprehensive guide to CCL syntax and functions

## Searching

### FindRowsIfContains

Returns the indices of rows that contain the given element.

```go
func (dt *DataTable) FindRowsIfContains(value any) []int
```

**Parameters:**

- `value`: Value to search for

**Returns:**

- `[]int`: Slice of row indices that contain the value

**Example:**

```go
rowIndices := dt.FindRowsIfContains("John")
```

### FindRowsIfContainsAll

Returns the indices of rows that contain all the given elements.

```go
func (dt *DataTable) FindRowsIfContainsAll(values ...any) []int
```

**Parameters:**

- `values`: Values to search for

**Returns:**

- `[]int`: Slice of row indices that contain all values

**Example:**

```go
rowIndices := dt.FindRowsIfContainsAll("John", 25)
```

### FindColsIfContains

Returns the indices of columns that contain the given element.

```go
func (dt *DataTable) FindColsIfContains(value any) []string
```

**Parameters:**

- `value`: Value to search for

**Returns:**

- `[]string`: Slice of column indices that contain the value

**Example:**

```go
colIndices := dt.FindColsIfContains("Engineer")
```

### FindColsIfContainsAll

Returns the indices of columns that contain all the given elements.

```go
func (dt *DataTable) FindColsIfContainsAll(values ...any) []string
```

**Parameters:**

- `values`: Values to search for

**Returns:**

- `[]string`: Slice of column indices that contain all the values

**Example:**

```go
colIndices := dt.FindColsIfContainsAll("Engineer", 25)
```

### FindRowsIfAnyElementContainsSubstring

Returns the indices of rows where at least one element contains the given substring.

```go
func (dt *DataTable) FindRowsIfAnyElementContainsSubstring(substring string) []int
```

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]int`: Slice of row indices where at least one element contains the substring

**Example:**

```go
rowIndices := dt.FindRowsIfAnyElementContainsSubstring("John")
```

### FindRowsIfAllElementsContainSubstring

Returns the indices of rows where all elements contain the given substring.

```go
func (dt *DataTable) FindRowsIfAllElementsContainSubstring(substring string) []int
```

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]int`: Slice of row indices where all elements contain the substring

**Example:**

```go
rowIndices := dt.FindRowsIfAllElementsContainSubstring("data")
```

### FindColsIfAnyElementContainsSubstring

Returns the indices of columns where at least one element contains the given substring.

```go
func (dt *DataTable) FindColsIfAnyElementContainsSubstring(substring string) []string
```

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]string`: Slice of column indices where at least one element contains the substring

**Example:**

```go
colIndices := dt.FindColsIfAnyElementContainsSubstring("Engineer")
```

### FindColsIfAllElementsContainSubstring

Returns the indices of columns where all elements contain the given substring.

```go
func (dt *DataTable) FindColsIfAllElementsContainSubstring(substring string) []string
```

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]string`: Slice of column indices where all elements contain the substring

**Example:**

```go
colIndices := dt.FindColsIfAllElementsContainSubstring("data")
```

## Filtering

### Filter

Filters the DataTable using a custom filter function.

```go
func (dt *DataTable) Filter(filterFunc FilterFunc) *DataTable
```

**Parameters:**

- `filterFunc`: Custom filter function

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
    return value != nil
})
```

### FilterByCustomElement

Filters the DataTable based on a custom function applied to each element.

```go
func (dt *DataTable) FilterByCustomElement(f func(value any) bool) *DataTable
```

**Parameters:**

- `f`: Custom filter function that takes a value and returns a boolean

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
// Filter to keep only numeric values greater than 10
filtered := dt.FilterByCustomElement(func(value any) bool {
    if num, ok := value.(float64); ok {
        return num > 10
    }
    return false
})
```

### FilterByColNameEqualTo

Filters columns by exact name match.

```go
func (dt *DataTable) FilterByColNameEqualTo(columnName string) *DataTable
```

**Parameters:**

- `columnName`: Column name to match

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColNameEqualTo("age")
```

### FilterByColIndexGreaterThan

Filters columns by index greater than the specified threshold.

```go
func (dt *DataTable) FilterByColIndexGreaterThan(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColIndexGreaterThan("B") // Columns C, D, E...
```

### FilterByColIndexGreaterThanOrEqualTo

Filters columns by index greater than or equal to the specified threshold.

```go
func (dt *DataTable) FilterByColIndexGreaterThanOrEqualTo(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColIndexGreaterThanOrEqualTo("B") // Columns B, C, D...
```

### FilterByColIndexLessThan

Filters columns by index less than the specified threshold.

```go
func (dt *DataTable) FilterByColIndexLessThan(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColIndexLessThan("C") // Columns A, B
```

### FilterByColIndexLessThanOrEqualTo

Filters columns by index less than or equal to the specified threshold.

```go
func (dt *DataTable) FilterByColIndexLessThanOrEqualTo(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColIndexLessThanOrEqualTo("C") // Columns A, B, C
```

### FilterByColIndexEqualTo

Filters columns by exact index match.

```go
func (dt *DataTable) FilterByColIndexEqualTo(index string) *DataTable
```

**Parameters:**

- `index`: Column index (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColIndexEqualTo("B") // Only column B
```

### FilterByColNameContains

Filters columns whose name contains the specified substring.

```go
func (dt *DataTable) FilterByColNameContains(substring string) *DataTable
```

**Parameters:**

- `substring`: Substring to search for in column names

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByColNameContains("age") // Columns with "age" in name
```

### FilterByRowNameEqualTo

Filters rows by exact name match.

```go
func (dt *DataTable) FilterByRowNameEqualTo(name string) *DataTable
```

**Parameters:**

- `name`: Row name to match

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowNameEqualTo("John")
```

### FilterByRowNameContains

Filters rows whose name contains the specified substring.

```go
func (dt *DataTable) FilterByRowNameContains(substring string) *DataTable
```

**Parameters:**

- `substring`: Substring to search for in row names

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowNameContains("John") // Rows with "John" in name
```

### FilterByRowIndexGreaterThan

Filters rows by index greater than the specified threshold.

```go
func (dt *DataTable) FilterByRowIndexGreaterThan(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowIndexGreaterThan(5) // Rows 6, 7, 8...
```

### FilterByRowIndexGreaterThanOrEqualTo

Filters rows by index greater than or equal to the specified threshold.

```go
func (dt *DataTable) FilterByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowIndexGreaterThanOrEqualTo(5) // Rows 5, 6, 7...
```

### FilterByRowIndexLessThan

Filters rows by index less than the specified threshold.

```go
func (dt *DataTable) FilterByRowIndexLessThan(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowIndexLessThan(5) // Rows 0, 1, 2, 3, 4
```

### FilterByRowIndexLessThanOrEqualTo

Filters rows by index less than or equal to the specified threshold.

```go
func (dt *DataTable) FilterByRowIndexLessThanOrEqualTo(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowIndexLessThanOrEqualTo(5) // Rows 0, 1, 2, 3, 4, 5
```

### FilterByRowIndexEqualTo

Filters rows by exact index match.

```go
func (dt *DataTable) FilterByRowIndexEqualTo(index int) *DataTable
```

**Parameters:**

- `index`: Row index (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterByRowIndexEqualTo(3) // Only row 3
```

## Statistical Analysis

### Summary

Displays a comprehensive statistical summary of the DataTable.

```go
func (dt *DataTable) Summary()
```

**Example:**

```go
dt.Summary() // Displays summary to console
```

### Size

Returns the dimensions of the DataTable.

```go
func (dt *DataTable) Size() (numRows int, numCols int)
```

**Returns:**

- `numRows`: Number of rows
- `numCols`: Number of columns

**Example:**

```go
rows, cols := dt.Size()
fmt.Printf("Table has %d rows and %d columns\n", rows, cols)
```

### Mean

Calculates the mean of all numeric values in the DataTable.

```go
func (dt *DataTable) Mean() any
```

**Returns:**

- `any`: Mean value of all numeric data

**Example:**

```go
mean := dt.Mean()
fmt.Printf("Overall mean: %v\n", mean)
```

## Utility Methods

### Show

Displays the DataTable content in the console.

```go
func (dt *DataTable) Show()
```

**Example:**

```go
dt.Show() // Display table content in console
```

### ShowRange

Displays the DataTable with a specified range of rows.

```go
func (dt *DataTable) ShowRange(startEnd ...interface{})
```

**Parameters:**

- `startEnd`: Optional parameters to specify the range of rows to display:
  - No parameters: shows all rows
  - One positive value: shows first N rows (e.g., ShowRange(5) shows first 5 rows)
  - One negative value: shows last N rows (e.g., ShowRange(-5) shows last 5 rows)
  - Two values [start, end]: shows rows from index start (inclusive) to index end (exclusive)
  - If end is nil: shows rows from index start to the end of the table

**Example:**

```go
dt.ShowRange()      // Show all rows
dt.ShowRange(5)     // Show first 5 rows
dt.ShowRange(-5)    // Show last 5 rows
dt.ShowRange(2, 10) // Show rows 2-9
dt.ShowRange(2, nil) // Show rows from index 2 to end
```

### ShowTypes

Displays the data types of each column.

```go
func (dt *DataTable) ShowTypes()
```

**Example:**

```go
dt.ShowTypes() // Display column type information
```

### ShowTypesRange

Displays the data types of columns within a specified range.

```go
func (dt *DataTable) ShowTypesRange(startEnd ...interface{})
```

**Parameters:**

- `startEnd`: Optional parameters to specify the range of rows to display type information for

**Example:**

```go
dt.ShowTypesRange(5)     // Show types for first 5 rows
dt.ShowTypesRange(-5)    // Show types for last 5 rows
dt.ShowTypesRange(2, 10) // Show types for rows 2-9
```

### GetName

Gets the DataTable name.

```go
func (dt *DataTable) GetName() string
```

**Returns:**

- `string`: DataTable name

**Example:**

```go
name := dt.GetName()
fmt.Printf("Table name: %s\n", name)
```

### SetName

Sets the DataTable name.

```go
func (dt *DataTable) SetName(name string) *DataTable
```

**Parameters:**

- `name`: New name

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
dt.SetName("updated_data")
```

### Map

Applies a transformation function to all elements in the DataTable and returns a new DataTable with the transformed results. The function provides access to row index, column index, and element value for context-aware transformations.

```go
func (dt *DataTable) Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable
```

**Parameters:**

- `mapFunc`: Transformation function that takes three parameters:
  - `rowIndex` (int): Zero-based row index
  - `colIndex` (string): Column index (A, B, C, ...)
  - `element` (any): The current element value
  - Returns: Transformed value of any type

**Returns:**

- `*DataTable`: A new DataTable with the transformed values

**Features:**

- **Index-aware transformations**: Access both row and column positions
- **Error recovery**: If transformation fails for any element, the original value is preserved
- **Structure preservation**: The new DataTable maintains the same structure and row names
- **Type flexibility**: Can transform between different data types

**Example:**

```go
// Example 1: Add position information to each element
mappedDt := dt.Map(func(rowIndex int, colIndex string, element any) any {
    return fmt.Sprintf("R%d_%s_%v", rowIndex, colIndex, element)
})

// Example 2: Apply different transformations based on column
conditionalDt := dt.Map(func(rowIndex int, colIndex string, element any) any {
    switch colIndex {
    case "A": // First column
        if num, ok := element.(int); ok {
            return num * 10 // Multiply by 10
        }
    case "B": // Second column
        if num, ok := element.(int); ok {
            return num + rowIndex // Add row index
        }
    default:
        return fmt.Sprintf("Col_%s_Row_%d", colIndex, rowIndex)
    }
    return element
})

// Example 3: Mixed data type handling with index logic
transformedDt := dt.Map(func(rowIndex int, colIndex string, element any) any {
    // Apply transformation only to even rows in column A
    if colIndex == "A" && rowIndex%2 == 0 {
        if str, ok := element.(string); ok {
            return strings.ToUpper(str)
        }
        if num, ok := element.(int); ok {
            return num * 2
        }
    }
    return element
})
```

### Transpose

Transposes the DataTable (rows become columns and vice versa).

```go
func (dt *DataTable) Transpose() *DataTable
```

**Returns:**

- `*DataTable`: The transposed DataTable

**Example:**

```go
transposed := dt.Transpose()
```

## Notes

1. **Type Safety**: DataTable uses the `any` type to store data. Please ensure proper type conversion when operating on the data.

2. **Memory Management**: For large datasets, consider using streaming or batch processing to avoid memory overflow.

3. **Error Handling**: Most methods return errors. Please handle these errors appropriately to ensure program stability.

4. **Concurrency**: DataTable has built-in mutex protection for concurrent access, but complex operations may still require additional synchronization.

5. **Filter Operations**: Filter operations create new DataTable instances; original data is not modified.

6. **SQL Operations**: When using SQL-related functionality, ensure proper database connection configuration and permissions. The ToSQL method uses GORM, while ReadSQL uses standard database/sql.

7. **Column Indexing**: Columns can be accessed by both alphabetical indices (A, B, C...) and numeric indices (0, 1, 2...).

8. **Method Chaining**: Many methods return `*DataTable` to support method chaining for fluent API usage.
