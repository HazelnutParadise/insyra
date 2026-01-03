# DataTable

DataTable is the core data structure of Insyra for handling structured data. It provides rich data manipulation functionality including reading, writing, filtering, statistical analysis, and transformation operations.

## Table of Contents

- [Data Structure](#data-structure)
- [Creating DataTable](#creating-datatable)
- [Data Loading](#data-loading)
- [Data Saving](#data-saving)
- [Data Operations](#data-operations)
  - [Merge](#merge)
- [Data Replacement](#data-replacement)
- [Column Calculation](#column-calculation)
- [Searching](#searching)
- [Filtering](#filtering)
- [Statistical Analysis](#statistical-analysis)
- [Utility Methods](#utility-methods)
- [AtomicDo](#atomicdo)
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

    // AtomicDo support (actor-style serialization)
    cmdCh    chan func()
    initOnce sync.Once
    closed   atomic.Bool
}
```

**Field Descriptions:**

- `columns`: Slice of DataList pointers representing table columns
- `columnIndex`: Maps column indices (A, B, C...) to slice positions
- `rowNames`: Maps row names to their indices
- `name`: Name of the DataTable
- `creationTimestamp`: Unix timestamp when the table was created
- `lastModifiedTimestamp`: Unix timestamp when the table was last modified
- `cmdCh`, `initOnce`, `closed`: Internal fields enabling `AtomicDo` actor-style, serialized execution for thread-safety without external locks

### Naming Conventions

- **Table Names**: Use snake-style Pascal case (e.g., `Factor_Loadings`, `Communalities`) to avoid spelling errors caused by spaces.
- **Column Names**: Follow snake-style Pascal case for consistency.
- **Row Names**: Follow snake-style Pascal case for consistency.

## AtomicDo

`AtomicDo` provides safe, serialized access to a DataTable via an internal actor goroutine. All operations inside the function run in order and without races, allowing concurrent callers to compose multi-step updates safely.

```go
func (dt *DataTable) AtomicDo(f func(*DataTable))
```

- Single-threaded execution: `AtomicDo` tasks run one at a time.
- Reentrant: Calls made within `AtomicDo` run immediately (no deadlock).
- Cross-object nesting: Calling `dl.AtomicDo` from within `dt.AtomicDo` (and vice versa) is supported by inline execution to avoid deadlocks.
- Closed behavior: After `dt.Close()`, `AtomicDo` executes the function inline without scheduling.

Examples

- Append rows and update indices atomically:

```go
dt.AtomicDo(func(dt *insyra.DataTable) {
    dt.AppendRowsByColName(map[string]any{"name": "Alice", "age": 25})
    // Accessing and updating structures here is serialized
    _ = dt.Size()
})
```

- Concurrent writers without locks:

```go
wg := sync.WaitGroup{}
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(idx int) {
        defer wg.Done()
        dt.AtomicDo(func(dt *insyra.DataTable) {
            dt.UpdateElement(idx, "A", idx)
        })
    }(i)
}
wg.Wait()
```

Guidelines

- Keep the function body short; avoid blocking I/O or long CPU work inside `AtomicDo`.
- Prepare data outside; perform minimal mutations inside `AtomicDo`.
- Use `AtomicDo` for sequences that must observe consistent state across columns/rows.

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

### ReadCSV_File

Reads a CSV file and loads the data into a new DataTable.

```go
func ReadCSV_File(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)
```

**Parameters:**

- `filePath`: CSV file path
- `setFirstColToRowNames`: Whether to use the first column as row names
- `setFirstRowToColNames`: Whether to use the first row as column names

**Returns:**

- `*DataTable`: New DataTable with loaded data
- `error`: Error information, returns nil if successful

**Example:**

```go
dt, err := insyra.ReadCSV_File("data.csv", false, true)
if err != nil {
    log.Fatal(err)
}
```

### ReadCSV_String

Reads CSV data from a string and loads it into a new DataTable.

```go
func ReadCSV_String(csvString string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)
```

**Parameters:**

- `csvString`: CSV data as string
- `setFirstColToRowNames`: Whether to use the first column as row names
- `setFirstRowToColNames`: Whether to use the first row as column names

**Returns:**

- `*DataTable`: New DataTable with loaded data
- `error`: Error information, returns nil if successful

**Example:**

```go
csvData := "name,age,city\nJohn,30,NYC\nJane,25,LA"
dt, err := insyra.ReadCSV_String(csvData, false, true)
if err != nil {
    log.Fatal(err)
}
```

### ReadJSON_File

Reads a JSON file and loads the data into a new DataTable.

```go
func ReadJSON_File(filePath string) (*DataTable, error)
```

**Parameters:**

- `filePath`: JSON file path

**Returns:**

- `*DataTable`: New DataTable with loaded data
- `error`: Error information, returns nil if successful

**Example:**

```go
dt, err := insyra.ReadJSON_File("data.json")
if err != nil {
    log.Fatal(err)
}
```

### ReadJSON

Reads JSON data (supports bytes, string, slice, map, or any JSON-compatible value) and loads it into a new DataTable.

```go
func ReadJSON(data any) (*DataTable, error)
```

**Parameters:**

- `data`: JSON input (e.g., []byte, string, []map[string]any, map[string]any, etc.)

**Returns:**

- `*DataTable`: New DataTable with loaded data
- `error`: Error information, returns nil if successful

**Example:**

```go
jsonData := []byte(`[{"name":"John","age":30},{"name":"Jane","age":25}]`)
dt, err := insyra.ReadJSON(jsonData)
if err != nil {
    log.Fatal(err)
}
```

### Slice2DToDataTable

Converts a 2D slice of any type into a DataTable. Supports various 2D array types including `[][]any`, `[][]int`, `[][]int64`, `[][]float32`, `[][]float64`, `[][]string`, and more.

```go
func Slice2DToDataTable(data any) (*DataTable, error)
```

**Alias:** `ReadSlice2D` — provided as a convenience alias for `Slice2DToDataTable`.

**Quick example using the alias:**

```go
// Using the alias
dt, err := insyra.ReadSlice2D([][]any{
    {1, "Alice", 3.5},
    {2, "Bob", 4.0},
})
if err != nil {
    log.Fatal(err)
}
```

**Parameters:**

- `data`: A 2D slice/array of any type (e.g., `[][]any`, `[][]int64`, `[][]float64`, `[][]string`)

**Returns:**

- `*DataTable`: New DataTable with converted data (nil if error occurs)
- `error`: Error information, returns nil if successful

**Error Cases:**

The function returns an error for:

- `nil` input data
- Empty 2D slice
- Input is not a 2D slice (e.g., 1D slice or non-slice type)
- First row is empty
- Any row is not a slice or array

**Supported Types:**

The function uses reflection to support any 2D array type:

- `[][]any` - Mixed types
- `[][]int`, `[][]int32`, `[][]int64` - Integer types
- `[][]float32`, `[][]float64` - Floating-point types
- `[][]string` - String type
- `[][]bool` - Boolean type
- Any other type convertible to `any`

**Features:**

- **Type Conversion**: Automatically converts various types to `any` for storage
- **Inconsistent Row Lengths**: Uses the number of columns from the first row; missing columns are filled with `nil`
- **Comprehensive Error Handling**: Returns descriptive error messages for invalid input

**Examples:**

```go
// Example 1: Convert [][]any
data1 := [][]any{
    {1, "Alice", 3.5},
    {2, "Bob", 4.0},
    {3, "Charlie", 2.8},
}
dt1, err := insyra.Slice2DToDataTable(data1)
if err != nil {
    log.Fatal(err)
}

// Example 2: Convert [][]int64
data2 := [][]int64{
    {1, 2, 3},
    {4, 5, 6},
    {7, 8, 9},
}
dt2, err := insyra.Slice2DToDataTable(data2)
if err != nil {
    log.Fatal(err)
}

// Example 3: Convert [][]float64
data3 := [][]float64{
    {1.1, 2.2, 3.3},
    {4.4, 5.5, 6.6},
    {7.7, 8.8, 9.9},
}
dt3, err := insyra.Slice2DToDataTable(data3)
if err != nil {
    log.Fatal(err)
}

// Example 4: Convert [][]string
data4 := [][]string{
    {"Alice", "Bob", "Charlie"},
    {"Denver", "New York", "San Francisco"},
    {"Engineer", "Manager", "Developer"},
}
dt4, err := insyra.Slice2DToDataTable(data4)
if err != nil {
    log.Fatal(err)
}

// Example 5: Handling inconsistent row lengths
data5 := [][]any{
    {1, "Alice", 3.5},
    {2, "Bob"},              // Missing third column
    {3, "Charlie", 2.8, "Extra"}, // Extra column
}
dt5, err := insyra.Slice2DToDataTable(data5)
if err != nil {
    log.Fatal(err)
}
// Columns based on first row (3 columns)
// Second row: {2, "Bob", nil}
// Third row: {3, "Charlie", 2.8}

// Example 6: Error handling
_, err := insyra.Slice2DToDataTable(nil)
if err != nil {
    fmt.Println(err) // Output: input data cannot be nil
}

_, err = insyra.Slice2DToDataTable([][]any{})
if err != nil {
    fmt.Println(err) // Output: input data cannot be empty
}
```

### ReadSQL

Loads data from a database table or custom SQL query into a DataTable.

```go
func ReadSQL(db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error)
```

**Parameters:**

- `db`: GORM database connection
- `tableName`: Name of the database table to read from
- `options`: Optional configuration for reading data (ReadSQLOptions struct)

**ReadSQLOptions:**

- `RowNameColumn`: Column name to use as row names (default: "row_name")
- `Query`: Custom SQL query string (if provided, other options are ignored)
- `Limit`: Maximum number of rows to read (0 means no limit)
- `Offset`: Starting row offset
- `WhereClause`: WHERE clause for filtering
- `OrderBy`: ORDER BY clause for sorting

**Returns:**

- `*DataTable`: DataTable loaded with data
- `error`: Error information, returns nil if successful

**Example:**

```go
// Using table name with options
db, _ := gorm.Open(mysql.Open("connection_string"), &gorm.Config{})
dt, err := insyra.ReadSQL(db, "users", insyra.ReadSQLOptions{Limit: 100, OrderBy: "id DESC"})

// Using custom query
dt, err := insyra.ReadSQL(db, "", insyra.ReadSQLOptions{Query: "SELECT * FROM users WHERE active = 1"})
```

### ReadExcelSheet

Reads a specific sheet from an Excel file and loads the data into a new DataTable.

```go
func ReadExcelSheet(filePath string, sheetName string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)
```

**Parameters:**

- `filePath`: The path to the Excel file.
- `sheetName`: The name of the sheet to read.
- `setFirstColToRowNames`: If `true`, the first column of the sheet will be used as the row names for the DataTable.
- `setFirstRowToColNames`: If `true`, the first row of the sheet will be used as the column names for the DataTable.

**Returns:**

- `*DataTable`: A new DataTable containing the data from the Excel sheet.
- `error`: An error if the file cannot be opened, the sheet doesn't exist, or the data cannot be processed.

**Example:**

```go
dt, err := insyra.ReadExcelSheet("data.xlsx", "Sheet1", true, true)
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

### ToJSON_String

Converts the DataTable to JSON format and returns it as a string.

```go
func (dt *DataTable) ToJSON_String(useColNames bool) string
```

**Parameters:**

- `useColNames`: Whether to use column names as keys in JSON objects

**Returns:**

- `string`: JSON data as a string

**Example:**

```go
jsonStr := dt.ToJSON_String(true)
fmt.Println(jsonStr)
```

### ToMap

Alias for `Data()`. Returns the table as a map from column index/name to the column data slice.

```go
func (dt *DataTable) ToMap(useNamesAsKeys ...bool) map[string][]any
```

**Parameters:**

- `useNamesAsKeys` (optional): When true, use column names as keys in the returned map; otherwise use generated column indices (A, B, ...). Default is false.

**Returns:**

- `map[string][]any`: A map where each key is a column index or name and the value is the column data slice.

**Example:**

```go
m := dt.ToMap(true) // use column names as keys when available
for k, col := range m {
    fmt.Println("Column:", k, "Length:", len(col))
}
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

### Merge

Merges two DataTables based on a key column or row name.

```go
func (dt *DataTable) Merge(other *DataTable, direction MergeDirection, mode MergeMode, on ...string) (*DataTable, error)
```

**Parameters:**

- `other`: The other DataTable to merge with.
- `direction`: The direction of the merge.
  - `insyra.MergeDirectionHorizontal`: Join columns side-by-side (like a SQL JOIN).
  - `insyra.MergeDirectionVertical`: Join rows top-to-bottom (matching columns by name).
- `mode`: The merge mode.
  - `insyra.MergeModeInner`: Only keep matches.
  - `insyra.MergeModeOuter`: Keep all data, filling missing parts with `nil`.
- `on`: (Optional) The name of the column to join on (for horizontal merge).
  - If `on` is omitted or an empty string (`""`) in a horizontal merge, the tables will be joined based on their **row names**.
  - For vertical merge, this parameter is ignored.

**Returns:**

- `*DataTable`: A new DataTable containing the merged data.
- `error`: An error if the key is not found or the mode/direction is invalid.

**Example (Horizontal Merge):**

```go
dt1 := insyra.NewDataTable(
    insyra.NewDataList("A", "B", "C").SetName("ID"),
    insyra.NewDataList(1, 2, 3).SetName("Val1"),
)
dt2 := insyra.NewDataTable(
    insyra.NewDataList("B", "C", "D").SetName("ID"),
    insyra.NewDataList(10, 20, 30).SetName("Val2"),
)

// Inner Join on ID
res, _ := dt1.Merge(dt2, insyra.MergeDirectionHorizontal, insyra.MergeModeInner, "ID")
// Result:
// ID, Val1, Val2
// B, 2, 10
// C, 3, 20

// Join by Row Names (on is omitted)
res, _ = dt1.Merge(dt2, insyra.MergeDirectionHorizontal, insyra.MergeModeInner)
```

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

Updates the value at a specific position. Returns the table to support chaining calls.

```go
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value any) *DataTable
```

**Parameters:**

- `rowIndex`: Row index (0-based)
- `columnIndex`: Column index (A, B, C...)
- `value`: New value

**Example:**

```go
// single call
dt.UpdateElement(0, "A", "Jane")

// chaining example
newCol := insyra.NewDataList(1, 2, 3, 4)
dt.UpdateElement(0, "A", "Jane").UpdateCol("B", newCol)
```

### UpdateCol

Updates an entire column with new data. Returns the table to support chaining calls.

```go
func (dt *DataTable) UpdateCol(index string, dl *DataList) *DataTable
```

**Parameters:**

- `index`: Column index (A, B, C...)
- `dl`: New DataList to replace the column

**Example:**

```go
newCol := insyra.NewDataList(1, 2, 3, 4)
// single call
dt.UpdateCol("A", newCol)

// chaining example
newRow := insyra.NewDataList("Jane", 28, "Manager")
dt.UpdateCol("A", newCol).UpdateRow(0, newRow)
```

### UpdateColByNumber

Updates an entire column by its numeric index. Returns the table to support chaining calls.

```go
func (dt *DataTable) UpdateColByNumber(index int, dl *DataList) *DataTable
```

**Parameters:**

- `index`: Numeric column index (0-based)
- `dl`: New DataList to replace the column

**Example:**

```go
newCol := insyra.NewDataList(1, 2, 3, 4)
// single call
dt.UpdateColByNumber(0, newCol)

// chaining example
newRow := insyra.NewDataList("Jane", 28, "Manager")
dt.UpdateColByNumber(0, newCol).UpdateRow(0, newRow)
```

### UpdateRow

Updates an entire row with new data. Returns the table to support chaining calls.

```go
func (dt *DataTable) UpdateRow(index int, dl *DataList) *DataTable
```

**Parameters:**

- `index`: Row index (0-based)
- `dl`: New DataList to replace the row

**Example:**

```go
newRow := insyra.NewDataList("Jane", 28, "Manager")
// single call
dt.UpdateRow(0, newRow)

// chaining example
newCol := insyra.NewDataList(1, 2, 3, 4)
dt.UpdateRow(0, newRow).UpdateCol("A", newCol)
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
func (dt *DataTable) GetRowNameByIndex(index int) (string, bool)
```

**Parameters:**

- `index`: The numeric index of the row (0-based). Negative indices are not supported for row names.

**Returns:**

- `string`: The name of the row. Returns an empty string if no name is set for this row.
- `bool`: `true` if a row name exists for this index, `false` otherwise.

**Example:**

```go
name, exists := dt.GetRowNameByIndex(0)
if exists {
    fmt.Printf("Name of the first row: %s\\n", name)
} else {
    fmt.Println("First row has no name")
}
```

### GetRowIndexByName

> [!NOTE]
> Since Insyra's Get methods usually support -1 as an index (representing the last element), always check the boolean return value to distinguish between "name not found" and "last row".

Gets the index of a row by its name. This is the inverse lookup of `GetRowNameByIndex`.

```go
func (dt *DataTable) GetRowIndexByName(name string) (int, bool)
```

**Parameters:**

- `name`: The name of the row to find.

**Returns:**

- `int`: The row index (0-based). Returns `-1` if the row name does not exist.
- `bool`: `true` if the row name exists, `false` otherwise.

**Notes:**

- Always check the boolean return value to confirm that the row name exists, especially since `-1` can also represent the last row in some Insyra methods.
- A log warning will be emitted if the row name is not found.

**Example:**

```go
index, exists := dt.GetRowIndexByName("FirstRow")
if exists {
    fmt.Printf("Row 'FirstRow' is at index: %d\\n", index)
    row := dt.GetRow(index)
    // Use the row...
} else {
    fmt.Println("Row 'FirstRow' not found")
}
```

### RowNamesToFirstCol

Moves all row names to the first column of the DataTable and clears the row names mapping.

```go
func (dt *DataTable) RowNamesToFirstCol() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// If DataTable has rows named "row1", "row2", "row3"
// After calling RowNamesToFirstCol(), these names will appear in the first column
dt.RowNamesToFirstCol()
```

### DropRowNames

Removes all row names from the DataTable.

```go
func (dt *DataTable) DropRowNames() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// Remove all row names
dt.DropRowNames()
```

### RowNames

Returns a slice containing all row names in order. Returns empty strings for rows without names.

```go
func (dt *DataTable) RowNames() []string
```

**Returns:**

- `[]string`: Slice of row names

**Example:**

```go
names := dt.RowNames()
for i, name := range names {
    if name != "" {
        fmt.Printf("Row %d: %s\n", i, name)
    }
}
```

### SetRowNames

Sets the row names of the DataTable using a slice of strings. Only sets names for existing rows; excess names are ignored.

```go
func (dt *DataTable) SetRowNames(rowNames []string) *DataTable
```

**Parameters:**

- `rowNames`: Slice of strings representing the new row names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetRowNames([]string{"Row1", "Row2", "Row3"}) // Set row names
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

### GetColNameByIndex

Gets the name of a column by its Excel-style index (A, B, C, ..., Z, AA, AB, ...).

```go
func (dt *DataTable) GetColNameByIndex(index string) string
```

**Parameters:**

- `index`: The Excel-style column index (e.g., "A", "B", "C", "AA", "AB").

**Returns:**

- `string`: The name of the column. Returns an empty string if the index is invalid or out of bounds.

**Example:**

```go
columnName := dt.GetColNameByIndex("A")
fmt.Printf("Name of column A: %s\\n", columnName)
```

### GetColNumberByName

Gets the numeric index of a column by its name.

```go
func (dt *DataTable) GetColNumberByName(name string) int
```

**Parameters:**

- `name`: The name of the column.

**Returns:**

- `int`: The numeric index of the column (0-based). Returns -1 if the column name is not found.

**Example:**

```go
index := dt.GetColNumberByName("column_name")
if index != -1 {
    fmt.Printf("Column index: %d\\n", index)
} else {
    fmt.Println("Column not found")
}
```

### GetColIndexByName

Gets the column index (A, B, C, ...) by its name.

```go
func (dt *DataTable) GetColIndexByName(name string) string
```

**Parameters:**

- `name`: The name of the column.

**Returns:**

- `string`: The column index (A, B, C, ...). Returns an empty string if the column name is not found.

**Example:**

```go
index := dt.GetColIndexByName("column_name")
fmt.Printf("Column index: %s\\n", index)
```

### GetColIndexByNumber

Gets the column index (A, B, C, ...) by its numeric index (0, 1, 2, ...).

```go
func (dt *DataTable) GetColIndexByNumber(number int) string
```

**Parameters:**

- `number`: The numeric column index (0-based). Negative values are supported and count from the end.

**Returns:**

- `string`: The column index (A, B, C, ...). Returns an empty string if the column number is out of bounds.

**Example:**

```go
index := dt.GetColIndexByNumber(0)
fmt.Printf("Column index: %s\\n", index)

// Get last column index
lastIndex := dt.GetColIndexByNumber(-1)
fmt.Printf("Last column index: %s\\n", lastIndex)
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

### SetColNames

Sets the column names of the DataTable using a slice of strings. If the slice has more elements than existing columns, new columns will be added. If the slice has fewer elements, excess columns will be set to empty names.

```go
func (dt *DataTable) SetColNames(colNames []string) *DataTable
```

**Parameters:**

- `colNames`: Slice of strings representing the new column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetColNames([]string{"Name", "Age", "Role"}) // Set column names
```

### SetHeaders

Alias for SetColNames, sets the column names of the DataTable.

```go
func (dt *DataTable) SetHeaders(headers []string) *DataTable
```

**Parameters:**

- `headers`: Slice of strings representing the new column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetHeaders([]string{"Name", "Age", "Role"}) // Set column names
```

### ColNamesToFirstRow

Moves all column names to the first row of data and then clears the column names.

```go
func (dt *DataTable) ColNamesToFirstRow() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// If DataTable has columns named "Name", "Age", "Role"
// After calling ColNamesToFirstRow(), the first row will contain ["Name", "Age", "Role"]
// and the column names will be cleared (set to empty strings)
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

### ColNames

Returns a slice containing all column names in order.

```go
func (dt *DataTable) ColNames() []string
```

**Returns:**

- `[]string`: Slice of column names

**Example:**

```go
names := dt.ColNames()
for i, name := range names {
    if name != "" {
        fmt.Printf("Column %d: %s\n", i, name)
    } else {
        fmt.Printf("Column %d: (unnamed)\n", i)
    }
}
```

### Headers

Alias for ColNames, returns a slice containing all column names in order.

```go
func (dt *DataTable) Headers() []string
```

**Returns:**

- `[]string`: Slice of column names

**Example:**

```go
headers := dt.Headers() // Same as dt.ColNames()
```

### DropColsByName

Drops columns by their names.

```go
func (dt *DataTable) DropColsByName(columnNames ...string) *DataTable
```

**Parameters:**

- `columnNames`: Names of columns to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsByName("Age", "Address")
```

### DropRowsByIndex

Drops rows by their numeric indices (0-based).

```go
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) *DataTable
```

**Parameters:**

- `rowIndices`: Numeric row indices (0-based) to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsByIndex(0, 2, 5) // Drops rows at indices 0, 2, and 5
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

Drops columns by their letter indices (e.g., "A", "B", "C").

```go
func (dt *DataTable) DropColsByIndex(columnIndices ...string) *DataTable
```

**Parameters:**

- `columnIndices`: Column letter indices (A, B, C...) to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsByIndex("A", "C") // Drops columns A and C
```

### DropColsByNumber

Drops columns by their numeric indices (0-based).

```go
func (dt *DataTable) DropColsByNumber(columnIndices ...int) *DataTable
```

**Parameters:**

- `columnIndices`: Numeric column indices (0-based) to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsByNumber(0, 2) // Drops first and third columns (columns at index 0 and 2)
```

### DropColsContainString

Drops columns that contain any string elements.

```go
func (dt *DataTable) DropColsContainString() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainString() // Drops all columns that have at least one string element
```

### DropColsContainNumber

Drops columns that contain any numeric elements.

```go
func (dt *DataTable) DropColsContainNumber() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNumber() // Drops all columns that have at least one numeric element
```

### DropColsContainNil

Drops columns that contain any nil (null) elements.

```go
func (dt *DataTable) DropColsContainNil() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNil() // Drops all columns that have at least one nil element
```

### DropColsContainNaN

Drops columns that contain any NaN (Not a Number) elements.

```go
func (dt *DataTable) DropColsContainNaN() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNaN() // Drops all columns that have at least one NaN element
```

### DropColsContain

Drops columns that contain the specified value(s).

```go
func (dt *DataTable) DropColsContain(value ...any) *DataTable
```

**Parameters:**

- `value`: The value(s) to check for in columns. Columns containing any of these values will be dropped.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Drop columns containing 0 or "N/A"
dt.DropColsContain(0, "N/A")
```

### DropColsContainExcelNA

Drops columns that contain Excel NA values ("#N/A").

```go
func (dt *DataTable) DropColsContainExcelNA() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Drop columns containing Excel NA values
dt.DropColsContainExcelNA()
```

### DropRowsContainString

Drops rows that contain any string elements.

```go
func (dt *DataTable) DropRowsContainString() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainString() // Drops all rows that have at least one string element
```

### DropRowsContainNumber

Drops rows that contain any numeric elements.

```go
func (dt *DataTable) DropRowsContainNumber() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNumber() // Drops all rows that have at least one numeric element
```

### DropRowsContainNil

Drops rows that contain any nil (null) elements.

```go
func (dt *DataTable) DropRowsContainNil() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNil() // Drops all rows that have at least one nil element
```

### DropRowsContainNaN

Drops rows that contain any NaN (Not a Number) elements.

```go
func (dt *DataTable) DropRowsContainNaN() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNaN() // Drops all rows that have at least one NaN element
```

### DropRowsContain

Drops rows that contain the specified value(s).

```go
func (dt *DataTable) DropRowsContain(value ...any) *DataTable
```

**Parameters:**

- `value`: The value(s) to check for in rows. Rows containing any of these values will be dropped.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Drop rows containing 0 or "N/A"
dt.DropRowsContain(0, "N/A")
```

### DropRowsContainExcelNA

Drops rows that contain Excel NA values ("#N/A").

```go
func (dt *DataTable) DropRowsContainExcelNA() *DataTable
```

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Drop rows containing Excel NA values
dt.DropRowsContainExcelNA()
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

### SimpleRandomSample

Performs simple random sampling on the DataTable.

```go
func (dt *DataTable) SimpleRandomSample(sampleSize int) *DataTable
```

**Parameters:**

- `sampleSize`: Number of rows to sample. If `sampleSize <= 0`, returns an empty DataTable. If `sampleSize >= number of rows`, returns a copy of the original DataTable.

**Returns:**

- `*DataTable`: A new DataTable containing the sampled rows

**Example:**

```go
// Sample 10 rows from the DataTable
sampled := dt.SimpleRandomSample(10)
```

**Notes:**

- Uses random permutation to ensure unbiased sampling
- Returns a new DataTable, leaving the original unchanged
- If sample size is greater than or equal to the number of rows, returns a full copy
- If sample size is less than or equal to 0, returns an empty DataTable

## Data Replacement

DataTable provides several methods to replace values within the entire table, a specific row, or a specific column.

### Replace

Replaces all occurrences of `oldValue` with `newValue` in the entire DataTable.

```go
func (dt *DataTable) Replace(oldValue, newValue any) *DataTable
```

### ReplaceNaNsWith

Replaces all occurrences of `NaN` with `newValue` in the entire DataTable.

```go
func (dt *DataTable) ReplaceNaNsWith(newValue any) *DataTable
```

### ReplaceNilsWith

Replaces all occurrences of `nil` with `newValue` in the entire DataTable.

```go
func (dt *DataTable) ReplaceNilsWith(newValue any) *DataTable
```

### ReplaceNaNsAndNilsWith

Replaces all occurrences of both `NaN` and `nil` with `newValue` in the entire DataTable.

```go
func (dt *DataTable) ReplaceNaNsAndNilsWith(newValue any) *DataTable
```

### ReplaceInRow

Replaces occurrences of `oldValue` with `newValue` in a specific row.

```go
func (dt *DataTable) ReplaceInRow(rowIndex int, oldValue, newValue any, mode ...int) *DataTable
```

**Parameters:**

- `rowIndex`: The index of the row.
- `oldValue`: The value to be replaced.
- `newValue`: The value to replace with.
- `mode` (optional):
  - `0` (default): Replace all occurrences.
  - `1`: Replace only the first occurrence.
  - `-1`: Replace only the last occurrence.

### ReplaceNaNsInRow / ReplaceNilsInRow / ReplaceNaNsAndNilsInRow

Similar to `ReplaceInRow`, but specifically for `NaN`, `nil`, or both.

```go
func (dt *DataTable) ReplaceNaNsInRow(rowIndex int, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNaNsAndNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable
```

### ReplaceInCol

Replaces occurrences of `oldValue` with `newValue` in a specific column.

```go
func (dt *DataTable) ReplaceInCol(colIndex string, oldValue, newValue any, mode ...int) *DataTable
```

**Parameters:**

- `colIndex`: The index or name of the column.
- `oldValue`: The value to be replaced.
- `newValue`: The value to replace with.
- `mode` (optional):
  - `0` (default): Replace all occurrences.
  - `1`: Replace only the first occurrence.
  - `-1`: Replace only the last occurrence.

### ReplaceNaNsInCol / ReplaceNilsInCol / ReplaceNaNsAndNilsInCol

Similar to `ReplaceInCol`, but specifically for `NaN`, `nil`, or both.

```go
func (dt *DataTable) ReplaceNaNsInCol(colIndex string, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNilsInCol(colIndex string, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNaNsAndNilsInCol(colIndex string, newValue any, mode ...int) *DataTable
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

// Row access (A.0 gets the first row of column A)
dt.AddColUsingCCL("base_value", "A.0")

// All columns reference (@.0 gets all columns of the first row)
dt.ExecuteCCL("NEW('first_row') = @.0")

// Aggregate functions (SUM, AVG, COUNT, MAX, MIN)
dt.AddColUsingCCL("total_sum", "SUM(A)")
dt.AddColUsingCCL("row_sum", "SUM(@.#)") // Row-wise sum
dt.AddColUsingCCL("table_sum", "SUM(@)") // Total table sum
dt.AddColUsingCCL("avg_val", "AVG(A + B)")
dt.AddColUsingCCL("count_val", "COUNT(@)") // Count non-nil cells
```

**Notes:**

- CCL allows performing calculations and operations on columns in a DataTable
- Column references use Excel-style notation (A, B, C...) for simplicity
- Supports mathematical operations, logical operations, conditionals and more
- See [CCL Documentation](CCL.md) for a comprehensive guide to CCL syntax and functions

### ExecuteCCL

Executes one or more CCL statements on the DataTable. Statements are separated by newlines and executed sequentially. Each statement sees the results of previous statements.

```go
func (dt *DataTable) ExecuteCCL(ccl string) *DataTable
```

**Parameters:**

- `ccl`: One or more CCL statements separated by newlines

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Sequential execution: col3 can use col1 and col2
dt.ExecuteCCL(`
    NEW('col1') = A * 2
    NEW('col2') = B + 10
    NEW('col3') = col1 + col2
    NEW('total') = SUM(@)
`)
```

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

Filters the DataTable using a custom filter function. Keeps only rows where the filter function returns true for at least one cell.

```go
func (dt *DataTable) Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable
```

**Parameters:**

- `filterFunc`: Custom filter function that receives:
  - `rowIndex`: Row index (0-based)
  - `columnIndex`: Column name
  - `value`: Cell value

  Returns `true` to keep the cell/row, `false` to discard

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
// Keep rows with non-nil values
filtered := dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
    return value != nil
})

// Keep rows where column "age" has values greater than 30
filtered := dt.Filter(func(rowIndex int, columnIndex string, value any) bool {
    if columnIndex == "age" {
        if num, ok := value.(int); ok {
            return num > 30
        }
    }
    return false
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

### FilterRows

Filters rows based on a custom function that checks each cell. Keeps only rows where the filter function returns true for at least one cell.

```go
func (dt *DataTable) FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable
```

**Parameters:**

- `filterFunc`: Custom filter function that receives:
  - `colIndex`: Column letter index (A, B, C...)
  - `colName`: Column name
  - `x`: Cell value

**Returns:**

- `*DataTable`: New filtered DataTable containing rows that match the filter condition

**Example:**

```go
// Keep rows that contain the value 100 in any column
filtered := dt.FilterRows(func(colIndex, colName string, x any) bool {
    return x == 100
})

// Keep rows where column A value is greater than 25
filtered := dt.FilterRows(func(colIndex, colName, x any) bool {
    return (colIndex == "A") && (x.(int) > 25)
})

// Keep rows where column named "age" value is greater than 25
filtered := dt.FilterRows(func(colIndex, colName, x any) bool {
    return (colName == "age") && (x.(int) > 25)
})
```

### FilterCols

Filters columns based on a custom function applied to each cell. Keeps only columns where the filter function returns true for at least one cell in that column.

```go
func (dt *DataTable) FilterCols(filterFunc func(rowIndex int, rowName string, x any) bool) *DataTable
```

**Parameters:**

- `filterFunc`: Custom filter function that receives:
  - `rowIndex`: index of the row (0-based)
  - `rowName`: row name (empty string if none)
  - `x`: cell value

**Returns:**

- `*DataTable`: New filtered DataTable containing columns that match the filter condition

**Examples:**

```go
// Keep columns that contain the value 4 in any row
filtered := dt.FilterCols(func(rowIndex int, rowName string, x any) bool {
    return x == 4
})

// Keep columns where the first row equals 1
filtered := dt.FilterCols(func(rowIndex int, rowName string, x any) bool {
    return (rowIndex == 0) && (x == 1)
})

// Keep columns where the row named "John" equals 4
// (requires setting row names first)
dt.SetRowNames([]string{"John", "Mary", "Bob"})
filtered := dt.FilterCols(func(rowIndex int, rowName string, x any) bool {
    return (rowName == "John") && (x == 4)
})
```

### FilterColsByColNameEqualTo

Filters columns by exact name match.

```go
func (dt *DataTable) FilterColsByColNameEqualTo(columnName string) *DataTable
```

**Parameters:**

- `columnName`: Column name to match

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColNameEqualTo("age")
```

### FilterColsByColIndexGreaterThan

Filters columns by index greater than the specified threshold.

```go
func (dt *DataTable) FilterColsByColIndexGreaterThan(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexGreaterThan("B") // Columns C, D, E...
```

### FilterColsByColIndexGreaterThanOrEqualTo

Filters columns by index greater than or equal to the specified threshold.

```go
func (dt *DataTable) FilterColsByColIndexGreaterThanOrEqualTo(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexGreaterThanOrEqualTo("B") // Columns B, C, D...
```

### FilterColsByColIndexLessThan

Filters columns by index less than the specified threshold.

```go
func (dt *DataTable) FilterColsByColIndexLessThan(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexLessThan("C") // Columns A, B
```

### FilterColsByColIndexLessThanOrEqualTo

Filters columns by index less than or equal to the specified threshold.

```go
func (dt *DataTable) FilterColsByColIndexLessThanOrEqualTo(threshold string) *DataTable
```

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexLessThanOrEqualTo("C") // Columns A, B, C
```

### FilterColsByColIndexEqualTo

Filters columns by exact index match.

```go
func (dt *DataTable) FilterColsByColIndexEqualTo(index string) *DataTable
```

**Parameters:**

- `index`: Column index (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexEqualTo("B") // Only column B
```

### FilterColsByColNameContains

Filters columns whose name contains the specified substring.

```go
func (dt *DataTable) FilterColsByColNameContains(substring string) *DataTable
```

**Parameters:**

- `substring`: Substring to search for in column names

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColNameContains("age") // Columns with "age" in name
```

### FilterRowsByRowNameEqualTo

Filters rows by exact name match.

```go
func (dt *DataTable) FilterRowsByRowNameEqualTo(name string) *DataTable
```

**Parameters:**

- `name`: Row name to match

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowNameEqualTo("John")
```

### FilterRowsByRowNameContains

Filters rows whose name contains the specified substring.

```go
func (dt *DataTable) FilterRowsByRowNameContains(substring string) *DataTable
```

**Parameters:**

- `substring`: Substring to search for in row names

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowNameContains("John") // Rows with "John" in name
```

### FilterRowsByRowIndexGreaterThan

Filters rows by index greater than the specified threshold.

```go
func (dt *DataTable) FilterRowsByRowIndexGreaterThan(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexGreaterThan(5) // Rows 6, 7, 8...
```

### FilterRowsByRowIndexGreaterThanOrEqualTo

Filters rows by index greater than or equal to the specified threshold.

```go
func (dt *DataTable) FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexGreaterThanOrEqualTo(5) // Rows 5, 6, 7...
```

### FilterRowsByRowIndexLessThan

Filters rows by index less than the specified threshold.

```go
func (dt *DataTable) FilterRowsByRowIndexLessThan(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexLessThan(5) // Rows 0, 1, 2, 3, 4
```

### FilterRowsByRowIndexLessThanOrEqualTo

Filters rows by index less than or equal to the specified threshold.

```go
func (dt *DataTable) FilterRowsByRowIndexLessThanOrEqualTo(threshold int) *DataTable
```

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexLessThanOrEqualTo(5) // Rows 0, 1, 2, 3, 4, 5
```

### FilterRowsByRowIndexEqualTo

Filters rows by exact index match.

```go
func (dt *DataTable) FilterRowsByRowIndexEqualTo(index int) *DataTable
```

**Parameters:**

- `index`: Row index (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexEqualTo(3) // Only row 3
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

### NumRows

Returns the number of rows in the DataTable.

```go
func (dt *DataTable) NumRows() int
```

**Returns:**

- `int`: Number of rows

**Example:**

```go
rows := dt.NumRows()
fmt.Printf("Table has %d rows\n", rows)
```

### NumCols

Returns the number of columns in the DataTable.

```go
func (dt *DataTable) NumCols() int
```

**Returns:**

- `int`: Number of columns

**Example:**

```go
cols := dt.NumCols()
fmt.Printf("Table has %d columns\n", cols)
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
func (dt *DataTable) ShowRange(startEnd ...any)
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
func (dt *DataTable) ShowTypesRange(startEnd ...any)
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

Sets the DataTable name. Use snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces.

```go
func (dt *DataTable) SetName(name string) *DataTable
```

**Parameters:**

- `name`: New name (recommended: snake-style Pascal case)

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
dt.SetName("Factor_Loadings")
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

### SortBy

Sorts the DataTable rows based on one or more column configurations. Supports multi-level sorting with stable sort.

```go
func (dt *DataTable) SortBy(configs ...DataTableSortConfig) *DataTable
```

**Parameters:**

- `configs`: Variable number of DataTableSortConfig structs specifying sort criteria

**DataTableSortConfig:**

```go
type DataTableSortConfig struct {
    ColumnIndex  string // Column index (A, B, C...)
    ColumnNumber int    // Column number (0-based)
    ColumnName   string // Column name
    Descending   bool   // Sort in descending order
}
```

**Returns:**

- `*DataTable`: The sorted DataTable (modified in-place)

**Description:**

- Supports sorting by column index, number, or name
- Multi-level sorting: sorts by the first config, then by subsequent configs for ties
- Uses stable sort to maintain relative order of equal elements
- At least one of ColumnIndex, ColumnNumber, or ColumnName must be specified

**Example:**

```go
// Single column sort
dt.SortBy(insyra.DataTableSortConfig{ColumnName: "Age", Descending: false})

// Multi-column sort: sort by Age ascending, then by Name descending
dt.SortBy(
    insyra.DataTableSortConfig{ColumnName: "Age", Descending: false},
    insyra.DataTableSortConfig{ColumnName: "Name", Descending: true},
)
```

### Clone

Creates a deep copy of the DataTable.

```go
func (dt *DataTable) Clone() *DataTable
```

**Returns:**

- `*DataTable`: A new DataTable instance that is a deep copy of the original

**Description:**

The Clone method creates a complete deep copy of the DataTable, including:

- All columns (each DataList is also cloned)
- Column index mapping
- Row names mapping
- Table name
- Creation timestamp (original timestamp is preserved)
- Last modified timestamp (reset to current time)

The cloned DataTable is completely independent of the original, so modifications to one will not affect the other.

**Example:**

```go
// Create original DataTable
original := NewDataTable()
original.AppendCols(NewDataList(1, 2, 3), NewDataList("a", "b", "c"))
original.SetName("Original Table")

// Clone the DataTable
cloned := original.Clone()

// Modify original - cloned remains unchanged
original.SetName("Modified Table")
original.columns[0].data[0] = 999

// cloned still has original name and data
fmt.Println(cloned.GetName()) // Output: Original Table
fmt.Println(cloned.columns[0].data[0]) // Output: 1
```

### To2DSlice

Converts the DataTable to a 2D slice of any type.

```go
func (dt *DataTable) To2DSlice() [][]any
```

**Returns:**

- `[][]any`: A 2D slice where each inner slice represents a row and contains the column values for that row

**Description:**

The To2DSlice method converts the DataTable into a standard Go 2D slice format. Each row in the DataTable becomes an inner slice in the returned 2D slice, and each column's value at that row position becomes an element in the inner slice.

- If a column is shorter than the maximum row count, `nil` values are used to fill the missing positions
- The returned slice is a deep copy of the data, so modifications to the slice won't affect the original DataTable
- This method is useful for interfacing with other Go libraries that expect 2D slice data structures

**Example:**

```go
// Create a DataTable with some data
dt := NewDataTable()
dt.AppendCols(
    NewDataList(1, 2, 3),        // Column A
    NewDataList("a", "b"),        // Column B (shorter)
    NewDataList("x", "y", "z", "w") // Column C (longer)
)

// Convert to 2D slice
slice := dt.To2DSlice()

// The result will be:
// [
//   [1, "a", "x"],
//   [2, "b", "y"],
//   [3, nil, "z"],
//   [nil, nil, "w"]
// ]

fmt.Printf("Number of rows: %d\n", len(slice))
fmt.Printf("Number of columns: %d\n", len(slice[0]))
fmt.Printf("Value at [0][0]: %v\n", slice[0][0])
```

## Notes

1. **Type Safety**: DataTable uses the `any` type to store data. Please ensure proper type conversion when operating on the data.

2. **Memory Management**: For large datasets, consider using streaming or batch processing to avoid memory overflow.

3. **Error Handling**: Some methods return `error` (e.g., file I/O, SQL operations). Many manipulation methods return `*DataTable` for chaining — check the specific function signatures and handle errors where applicable.

4. **Concurrency**: DataTable uses actor-style serialized execution via `AtomicDo` (internal command channel and goroutine) for thread-safety rather than a global mutex. Use `AtomicDo` for sequences that must observe consistent state and avoid long-blocking work inside it.

5. **Filter Operations**: Filter operations create new DataTable instances; original data is not modified.

6. **SQL Operations**: When using SQL-related functionality, ensure proper database connection configuration and permissions.

7. **Column Indexing**: Columns can be accessed by both alphabetical indices (A, B, C...) and numeric indices (0, 1, 2...).

8. **Method Chaining**: Many methods return `*DataTable` to support method chaining for fluent API usage.

9. **Naming Conventions**: Table names, column names, and row names should use snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces.
