# DataTable

DataTable is the core data structure of Insyra for handling structured data. It provides rich data manipulation functionality including reading, writing, filtering, statistical analysis, and transformation operations.

## Table of Contents

- [Data Structure](#data-structure)
- [Creating DataTable](#creating-datatable)
- [Data Loading](#data-loading)
- [Data Saving](#data-saving)
- [Data Operations](#data-operations)
  - [Merge](#merge)
  - [GroupBy](#groupby)
  - [Categorical Encoding](#categorical-encoding)
- [Data Replacement](#data-replacement)
- [Column Calculation](#column-calculation)
- [Searching](#searching)
- [Filtering](#filtering)
- [Statistical Analysis](#statistical-analysis)
- [Utility Methods](#utility-methods)
- [Error Handling](#error-handling)
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
    atomicActor core.AtomicActor
}
```

**Field Descriptions:**

- `columns`: Slice of DataList pointers representing table columns
- `columnIndex`: Maps column indices (A, B, C...) to slice positions
- `rowNames`: Maps row names to their indices
- `name`: Name of the DataTable
- `creationTimestamp`: Unix timestamp when the table was created
- `lastModifiedTimestamp`: Unix timestamp when the table was last modified
- `atomicActor`: Internal mutex + holder pair used by `AtomicDo` to serialise execution; same-goroutine re-entry runs inline without re-locking

### Naming Conventions

- **Table Names**: Use snake-style Pascal case (e.g., `Factor_Loadings`, `Communalities`) to avoid spelling errors caused by spaces.
- **Column Names**: Follow snake-style Pascal case for consistency.
- **Row Names**: Follow snake-style Pascal case for consistency.

## Creating DataTable

### NewDataTable

```go
func NewDataTable(columns ...*DataList) *DataTable
```

**Description:** Creates a new empty DataTable.

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

```go
func ReadCSV_File(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool, encoding ...string) (*DataTable, error)
```

**Description:** Reads a CSV file and loads the data into a new DataTable.

**Parameters:**

- `filePath`: CSV file path
- `setFirstColToRowNames`: Whether to use the first column as row names
- `setFirstRowToColNames`: Whether to use the first row as column names
- `encoding`: Optional encoding (`"auto"` by default); pass `"utf-8"`, `"big5"`, etc.

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

```go
func ReadCSV_String(csvString string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)
```

**Description:** Reads CSV data from a string and loads it into a new DataTable.

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

```go
func ReadJSON_File(filePath string) (*DataTable, error)
```

**Description:** Reads a JSON file and loads the data into a new DataTable.

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

```go
func ReadJSON(data any) (*DataTable, error)
```

**Description:** Reads JSON data (supports bytes, string, slice, map, or any JSON-compatible value) and loads it into a new DataTable.

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

```go
func Slice2DToDataTable(data any) (*DataTable, error)
```

**Description:** Converts a 2D slice of any type into a DataTable. Supports various 2D array types including `[][]any`, `[][]int`, `[][]int64`, `[][]float32`, `[][]float64`, `[][]string`, and more.

**Parameters:**

- `data`: A 2D slice/array of any type (e.g., `[][]any`, `[][]int64`, `[][]float64`, `[][]string`)

**Returns:**

- `*DataTable`: New DataTable with converted data (nil if error occurs)
- `error`: Error information, returns nil if successful

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

```go
func ReadSQL(db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error)
func ReadSQLContext(ctx context.Context, db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error)
func ReadSQLStream(ctx context.Context, db *gorm.DB, tableName string, options ...ReadSQLOptions) (<-chan ReadSQLChunk, error)
```

**Description:** Loads data from a database table or custom SQL query into a DataTable.

- `ReadSQL` is the simple form. It is equivalent to `ReadSQLContext(context.Background(), ...)`.
- `ReadSQLContext` is the context-aware variant. The query and row scanning run under `ctx`, so callers can cancel long-running reads.
- `ReadSQLStream` reads a (potentially huge) result set in chunks, emitting each chunk as a `*DataTable` on the returned channel. The channel closes when the stream completes, when `ctx` is cancelled, or after a fatal error. Use this when the result set does not fit in memory.

**Parameters:**

- `ctx`: Context that controls cancellation of the database calls (Context variants only)
- `db`: GORM database connection
- `tableName`: Name of the database table to read from. Ignored when `ReadSQLOptions.Query` is set.
- `options`: Optional configuration (`ReadSQLOptions` struct, see below)

**Returns:**

- `*DataTable`: DataTable loaded with data (single-result functions)
- `<-chan ReadSQLChunk`: Channel of streamed chunks (`ReadSQLStream`)
- `error`: Error information, returns nil if successful

**ReadSQLChunk:**

```go
type ReadSQLChunk struct {
    Table *DataTable // populated on success
    Err   error      // populated on failure or cancellation
}
```

Exactly one of `Table` or `Err` is set per chunk.

**ReadSQLOptions:**

| Field | Type | Description |
| --- | --- | --- |
| `RowNameColumn` | `string` | Column whose values become DataTable row names. Defaults to `"row_name"` when neither this nor `IndexCol` is set. |
| `IndexCol` | `string` | Alias for `RowNameColumn`; mirrors pandas' `read_sql(index_col=...)`. When non-empty, takes precedence. |
| `Query` | `string` | Custom SQL query. When set, all other query-shape options are ignored. |
| `Params` | `[]any` | Positional bind parameters for `Query`. Equivalent to pandas' `read_sql(params=...)`. |
| `Columns` | `[]string` | Restricts the auto-built `SELECT` to these columns. Ignored when `Query` is set. |
| `Schema` | `string` | Optional schema (PostgreSQL) or database (MySQL) prefix. SQLite ignores this. |
| `Limit` | `int` | Maximum number of rows to read (0 means no limit). |
| `Offset` | `int` | Starting row offset. |
| `WhereClause` | `string` | `WHERE` clause body (without the `WHERE` keyword). |
| `OrderBy` | `string` | `ORDER BY` clause body (without the `ORDER BY` keyword). |
| `ParseDates` | `[]string` | Columns whose `string`/`[]byte` values should be parsed as `time.Time`. Several common ISO-style layouts are tried in order (RFC3339, `2006-01-02 15:04:05`, etc.). |
| `DType` | `map[string]reflect.Type` | Forces the resulting Go type for the named columns. Recognized targets: `int*`/`uint*`/`float*`/`bool`/`string`/`time.Time`/`[]byte`. |
| `ChunkSize` | `int` | Per-chunk row count for `ReadSQLStream`. Defaults to 1000. Ignored by `ReadSQL`/`ReadSQLContext`. |

**Type handling:** When the driver returns numeric or boolean columns as `[]byte` (common with the MySQL driver), `ReadSQL` consults `rows.ColumnTypes()` and converts to the appropriate Go primitive. Binary columns (`BLOB`/`BYTEA`) are kept as `[]byte`. `DType` overrides take precedence over the default mapping.

**Examples:**

```go
// Simple read with options.
db, _ := gorm.Open(mysql.Open("connection_string"), &gorm.Config{})
dt, err := insyra.ReadSQL(db, "users", insyra.ReadSQLOptions{
    Limit:   100,
    OrderBy: "id DESC",
})

// Parameterized custom query.
dt, err := insyra.ReadSQL(db, "", insyra.ReadSQLOptions{
    Query:  "SELECT * FROM users WHERE active = ? AND age > ?",
    Params: []any{true, 18},
})

// Cancelable read.
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
dt, err := insyra.ReadSQLContext(ctx, db, "users")

// Streaming a huge table in chunks of 5000 rows.
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
ch, err := insyra.ReadSQLStream(ctx, db, "events", insyra.ReadSQLOptions{ChunkSize: 5000})
if err != nil {
    log.Fatal(err)
}
for chunk := range ch {
    if chunk.Err != nil {
        log.Fatal(chunk.Err)
    }
    process(chunk.Table)
}

// Force types and parse a date column.
dt, err := insyra.ReadSQL(db, "events", insyra.ReadSQLOptions{
    ParseDates: []string{"created_at"},
    DType: map[string]reflect.Type{
        "score": reflect.TypeFor[float64](),
    },
})
```

### ReadExcelSheet

```go
func ReadExcelSheet(filePath string, sheetName string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)
```

**Description:** Reads a specific sheet from an Excel file and loads the data into a new DataTable.

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

```go
func (dt *DataTable) ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error
```

**Description:** Saves the DataTable as a CSV file.

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

```go
func (dt *DataTable) ToJSON(filePath string, useColNames bool) error
```

**Description:** Saves the DataTable as a JSON file.

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

```go
func (dt *DataTable) ToJSON_Bytes(useColNames bool) []byte
```

**Description:** Converts the DataTable to JSON format and returns as bytes.

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

```go
func (dt *DataTable) ToJSON_String(useColNames bool) string
```

**Description:** Converts the DataTable to JSON format and returns it as a string.

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

```go
func (dt *DataTable) ToMap(useNamesAsKeys ...bool) map[string][]any
```

**Description:** Alias for `Data()`. Returns the table as a map from column index/name to the column data slice.

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

```go
func (dt *DataTable) ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error
func (dt *DataTable) ToSQLContext(ctx context.Context, db *gorm.DB, tableName string, options ...ToSQLOptions) error
```

**Description:** Writes the DataTable to a SQL database.

- `ToSQL` is the simple form. It is equivalent to `ToSQLContext(context.Background(), ...)`.
- `ToSQLContext` runs every database call under `ctx`, so callers can cancel long writes.

**Behavior:**

- All database operations (CREATE TABLE, ALTER TABLE, INSERT, etc.) are executed inside a single database transaction. If any step fails, the transaction is rolled back and no partial changes are committed.
- Rows are inserted with **batched multi-value INSERTs**. The default batch size is 500 rows; tune via `ToSQLOptions.BatchSize`. Note that `BatchSize × column-count` must stay below the driver's bind-parameter limit (PostgreSQL/MySQL: 65535).
- When `IfExists` is set to append mode, `ToSQL` fetches the existing table's columns once and adds any missing ones via `ALTER TABLE` before inserting (no per-row schema introspection).
- Type inference samples the first non-nil value in each column. Recognized types include `time.Time` (→ `DATETIME`/`TIMESTAMP`), `[]byte` (→ `BLOB`/`BYTEA`), `sql.Null*` (unwraps the underlying value), and pointers (dereferenced). All-nil columns fall back to `TEXT`.
- You can provide `ColumnTypes` to override the inferred SQL types.

**Parameters:**

- `ctx`: Context that controls cancellation of the database calls (`ToSQLContext` only)
- `db`: GORM database connection
- `tableName`: Target table name
- `options`: Optional SQL options (`ToSQLOptions` struct, see below)

**ToSQLOptions:**

| Field | Type | Description |
| --- | --- | --- |
| `IfExists` | `SQLActionIfTableExists` | Behavior when the target table already exists. See enum values below. Default: `SQLActionIfTableExistsFail`. |
| `RowNames` | `bool` | If true, include row names as a `row_name` column (default type `TEXT`). |
| `ColumnTypes` | `map[string]string` | Explicit SQL column types per column, used instead of type inference. |
| `Schema` | `string` | Optional schema (PostgreSQL) or database (MySQL) prefix. SQLite ignores this. The caller is responsible for any required quoting. |
| `BatchSize` | `int` | Rows per multi-value `INSERT`. Zero means use the package default (500). |

**SQLActionIfTableExists values:**

- `SQLActionIfTableExistsFail` — return an error if the table exists (default)
- `SQLActionIfTableExistsReplace` — drop and recreate the table
- `SQLActionIfTableExistsAppend` — keep the table and append rows, adding missing columns if needed

**Returns:**

- `error`: Error information, returns nil if successful; if any DDL/DML step fails, an error is returned and the entire operation is rolled back.

**Examples:**

```go
db, _ := gorm.Open(mysql.Open("connection_string"))

// Replace existing table if present.
err := dt.ToSQL(db, "users", ToSQLOptions{IfExists: SQLActionIfTableExistsReplace})
if err != nil {
    log.Fatal(err)
}

// Append into an existing table; missing columns are added once.
err = dt.ToSQL(db, "events", ToSQLOptions{
    IfExists:  SQLActionIfTableExistsAppend,
    BatchSize: 1000,
})

// Cancelable write with a deadline.
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err = dt.ToSQLContext(ctx, db, "events_archive", ToSQLOptions{
    IfExists: SQLActionIfTableExistsReplace,
    Schema:   "analytics",
})
```

## Data Operations

### Merge

```go
func (dt *DataTable) Merge(other *DataTable, direction MergeDirection, mode MergeMode, on ...string) (*DataTable, error)
```

**Description:** Merges two DataTables based on a key column or row name.

**Parameters:**

- `other`: The other DataTable to merge with.
- `direction`: The direction of the merge.
  - `insyra.MergeDirectionHorizontal`: Join columns side-by-side (like a SQL JOIN).
  - `insyra.MergeDirectionVertical`: Join rows top-to-bottom (matching columns by name).
- `mode`: The merge mode.
  - `insyra.MergeModeInner`: Only keep matches.
    - `insyra.MergeModeOuter`: Keep all data, filling missing parts with `nil`.
    - `insyra.MergeModeLeft`: Keep all rows/keys from the first (left) table and attach matching rows from the second table; non-matching fields from the second table will be `nil`.
    - `insyra.MergeModeRight`: Keep all rows/keys from the second (right) table and attach matching rows from the first table; non-matching fields from the first table will be `nil`.
- `on`: (Optional) One or two column names to join on (for horizontal merge).
  - If one name is provided, it is used for both tables (e.g. `"ID"`).
  - If two names are provided, the first is the key column in the left table and the second is the key column in the right table (e.g. `"left_id", "right_id"`).
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

// Left Join (keep all rows from dt1)
resLeft, _ := dt1.Merge(dt2, insyra.MergeDirectionHorizontal, insyra.MergeModeLeft, "ID")
// Result (left join):
// ID, Val1, Val2
// A, 1, <nil>
// B, 2, 10
// C, 3, 20

// Right Join (keep all rows from dt2)
resRight, _ := dt1.Merge(dt2, insyra.MergeDirectionHorizontal, insyra.MergeModeRight, "ID")
// Result (right join):
// ID, Val1, Val2
// B, 2, 10
// C, 3, 20
// D, <nil>, 30
```

### GroupBy

```go
func (dt *DataTable) GroupBy(keyCols ...string) *GroupedDataTable
func (g *GroupedDataTable) Aggregate(configs ...AggregateConfig) *DataTable
func (g *GroupedDataTable) AggregateAll(op AggregateOp) *DataTable
func (g *GroupedDataTable) Count() *DataTable
func (g *GroupedDataTable) Describe(options ...DescribeOptions) *DataTable
```

**Description:** Splits the DataTable into groups by one or more key columns and applies aggregate functions to each group ("split-apply-combine"). The result is a new DataTable with one row per unique key combination — key columns first (in `GroupBy` order), then the aggregate columns (in `Aggregate` order). Group order in the output follows the order in which each key combination is first seen during a single linear scan; `nil` keys form their own group, and `int(1)` is kept distinct from the string `"1"`.

**`AggregateConfig` fields:**

- `SourceCol`: The column to aggregate. Resolved by name first, then as an Excel-style index (`"A"`, `"B"`, ...). Required for every op except `OpCountAll`.
- `As`: Output column name. When empty it is auto-named `"<source>_<op>"` (e.g. `"revenue_sum"`).
- `Op`: One of the `AggregateOp` constants below.
- `Custom`: Used only when `Op == OpCustom`. The provided function receives a `*DataList` containing the source-column values of the group, in original row order, including `nil` entries.

**Supported `AggregateOp`:**

| Op | Behavior |
|---|---|
| `OpSum` | Sum of numeric values; non-numeric and `nil` skipped |
| `OpMean` | Arithmetic mean of numeric values |
| `OpMedian` | Median of numeric values |
| `OpMin` / `OpMax` | Numeric min / max |
| `OpCount` | Count of **non-nil** values |
| `OpCountAll` | Count of **all** rows in the group (group size) |
| `OpStdev` / `OpStdevP` | Sample / population standard deviation |
| `OpVar` / `OpVarP` | Sample / population variance |
| `OpFirst` / `OpLast` | First / last **non-nil** value in original row order |
| `OpNUnique` | Distinct non-nil value count |
| `OpCustom` | User-supplied `Custom func(group *DataList) any` |

**Errors:** Unknown key columns, unknown source columns, missing `Custom` for `OpCustom`, empty configs, and empty key lists are reported via the parent `dt.Err()` instance-level error. The aggregate output is still returned (with affected columns nil-filled or empty), so callers can inspect partial results and continue chaining.

**Describe:** `GroupBy(...).Describe()` returns one row per group. Key columns are emitted first, followed by flattened summary columns such as `revenue_count`, `revenue_mean`, `revenue_25%`, and `segment_unique` when `IncludeAll` is enabled. Group order follows the same first-seen order as `Aggregate`.

**Example (single key, multiple aggregates):**

```go
dt := insyra.NewDataTable(
    insyra.NewDataList("east", "east", "west", "west", "south").SetName("region"),
    insyra.NewDataList(100, 200, 50, 75, 300).SetName("revenue"),
    insyra.NewDataList(1, 2, 3, 4, 5).SetName("qty"),
)

report := dt.GroupBy("region").Aggregate(
    insyra.AggregateConfig{SourceCol: "revenue", Op: insyra.OpSum,  As: "total_rev"},
    insyra.AggregateConfig{SourceCol: "revenue", Op: insyra.OpMean, As: "avg_rev"},
    insyra.AggregateConfig{SourceCol: "qty",     Op: insyra.OpSum,  As: "total_qty"},
)
// report columns: region, total_rev, avg_rev, total_qty
// report rows:    east 300 150 3 / west 125 62.5 7 / south 300 300 5
```

**Example (multiple keys, custom aggregate):**

```go
weighted := dt.GroupBy("region", "product").Aggregate(
    insyra.AggregateConfig{SourceCol: "revenue", Op: insyra.OpSum},  // auto-named "revenue_sum"
    insyra.AggregateConfig{
        SourceCol: "price",
        As:        "wprice",
        Op:        insyra.OpCustom,
        Custom: func(group *insyra.DataList) any {
            // Custom receives the group's column slice as a DataList. nil
            // entries are preserved; you handle them.
            return group.Mean()
        },
    },
)
```

**Pipeline:** `GroupBy + Aggregate` composes naturally with `FilterRows` and `SortBy`:

```go
top := dt.
    FilterRows(func(_, _ string, x any) bool { return true /* ... */ }).
    GroupBy("region").
    Aggregate(insyra.AggregateConfig{SourceCol: "revenue", Op: insyra.OpSum, As: "total"}).
    SortBy(insyra.DataTableSortConfig{ColumnName: "total", Descending: true})
```

### Pivot / Unpivot (long ↔ wide reshape)

```go
func (dt *DataTable) Pivot(cfg PivotConfig) (*DataTable, error)
func (dt *DataTable) Unpivot(cfg UnpivotConfig) (*DataTable, error)
```

**Description:** `Pivot` reshapes long-form data into wide form (each unique value of `cfg.Columns` becomes a new column header, with cells filled from `cfg.Values` and rows keyed by `cfg.Index`). `Unpivot` is the inverse: each row is expanded into one output row per `ValueVar`, with the original column name written into the new `VarName` column and the cell value written into `ValueName`. Both methods return a fresh `*DataTable`; the receiver is not modified. On error the returned `*DataTable` is empty and carries the failure on its `Err()`, so chained calls remain safe.

**Column reference resolution (applies to every column-name field below — `Index`, `Columns`, `Values`, `IDVars`, `ValueVars`):** each token is matched against `column.name` first; if no column has that name, it falls back to the Excel-style alphabetic index (`"A"` → column 0, `"B"` → column 1, ..., `"AA"` → column 26, ...). The first row of data is **never** consulted — column headers live only on `column.name` (set via `SetName`, `SetColNames`, CSV/Excel `firstRow2ColNames=true`, etc.). Tokens that match neither a name nor a valid alphabetic index are an error.

**`PivotConfig` fields:**

- `Index`: Identifier columns; their unique combinations form output rows. At least one required. Each entry follows the resolution rule above.
- `Columns`: Column whose unique values become new column headers. Required. Same resolution rule.
- `Values`: Column supplying cell values. Required. Same resolution rule.
- `AggFunc`: Aggregator applied when an `(Index, Columns)` pair has duplicates. Recognised: `"sum"`, `"mean"` (alias `"avg"`), `"median"`, `"min"`, `"max"`, `"count"` (non-nil), `"countall"` (group size), `"stdev"` (alias `"std"`), `"stdevp"` (alias `"stdp"`), `"var"`, `"varp"`, `"first"`, `"last"`, `"nunique"`, `"custom"`. When empty, duplicate `(Index, Columns)` combinations are an error.
- `Custom`: Required when `AggFunc == "custom"`. Receives the cell's values as a `*DataList` in original row order, including nil entries.
- `FillNA`: Value placed in cells whose `(Index, Columns)` combination is absent (or whose aggregated value is nil). Default `nil`.
- `SortCols`: When `true`, generated columns are emitted in sorted order of the key value; when `false` (default), first-seen order is preserved.

**`UnpivotConfig` fields:**

- `IDVars`: Columns kept as-is (identifier columns). Same resolution rule.
- `ValueVars`: Columns to unpivot. Same resolution rule. When empty, all non-`IDVars` columns are unpivoted.
- `VarName`: Name of the new "variable" column. Defaults to `"variable"`. The string written into this column for each output row is the source column's `name` (or, if a column was unnamed, its Excel-style label).
- `ValueName`: Name of the new "value" column. Defaults to `"value"`.
- `DropNA`: When `true`, omits rows whose value is `nil` or `NaN`.

**Errors:** Missing/unknown columns, duplicate listings, overlap between `Index`/`Columns`/`Values` (Pivot) or `IDVars`/`ValueVars` (Unpivot), unknown `AggFunc`, and `(Index, Columns)` duplicates without an aggregator are all surfaced via the returned `error` and recorded on the returned table's `Err()`.

**Example — long to wide:**

```go
// Long input:
//   region | product | sales
//   APAC   | A       | 10
//   APAC   | B       | 20
//   EMEA   | A       | 30
wide, err := dt.Pivot(insyra.PivotConfig{
    Index:    []string{"region"},
    Columns:  "product",
    Values:   "sales",
    AggFunc:  "sum",  // optional; required if (region, product) has duplicates
    FillNA:   0,
    SortCols: true,
})
// wide:
//   region | A  | B
//   APAC   | 10 | 20
//   EMEA   | 30 | 0
```

**Example — wide to long:**

```go
// Wide input:
//   id | Q1 | Q2 | Q3
//   1  | 5  | 4  | 3
long, err := dt.Unpivot(insyra.UnpivotConfig{
    IDVars:    []string{"id"},
    ValueVars: []string{"Q1", "Q2", "Q3"},
    VarName:   "question",
    ValueName: "score",
})
// long:
//   id | question | score
//   1  | Q1       | 5
//   1  | Q2       | 4
//   1  | Q3       | 3
```

**Relationship to `GroupBy + Aggregate`:** `Pivot` is essentially `GroupBy(Index..., Columns).Aggregate(Values, AggFunc)` followed by spreading the `Columns` key out into headers. When you only need the grouped summary (one row per key, no header spreading), use `GroupBy + Aggregate` directly — it is simpler and produces the same intermediate structure.

### Categorical Encoding

```go
func (dt *DataTable) OneHotEncode(opts OneHotOptions) (*DataTable, *OneHotEncoder, error)
func (dt *DataTable) LabelEncode(opts LabelEncodeOptions) (*DataTable, *LabelEncoder, error)
func (dt *DataTable) OrdinalEncode(opts OrdinalEncodeOptions) (*DataTable, *OrdinalEncoder, error)
```

**Description:** Categorical encoding turns string or mixed-type category columns into numeric columns that can feed `stats.LinearRegression`, KNN, PCA, and clustering. Each method returns a fresh `*DataTable`; the receiver is not modified. The returned encoder stores the fitted category mapping and can `Transform` another table with the same schema, such as a test set or prediction batch.

Column references are resolved by column name first, then Excel-style index (`"A"`, `"B"`, ..., `"AA"`). Category identity uses both type and value, so `int(1)` and string `"1"` are distinct. Missing means `nil` or `NaN`. For one-hot encoding, two distinct categories that would generate the same indicator column name (for example `int(1)` and `"1"`, both `c_1`, or `nil` and the string `"<nil>"`) are rejected at fit time; rename a category or set a distinct `Prefix`/`Separator`.

**Policies:**

| Type | Values | Behavior |
|---|---|---|
| `NaNPolicy` | `NaNAsCategory`, `NaNError`, `NaNSkip` | Missing becomes its own category, errors, or is skipped (`nil` for label/ordinal; all-zero for one-hot). |
| `UnknownPolicy` | `UnknownIgnore`, `UnknownError`, `UnknownAsNew` | On `Transform`, unseen categories become all-zero/`nil`, error, or are encoded as new outputs for that call. `UnknownAsNew` extends only the returned table; the fitted encoder is left unchanged, so `Transform` is pure and safe to reuse. |
| `LabelSort` | `LabelSortFirstSeen`, `LabelSortLexicographic`, `LabelSortByFrequency` | Controls label id assignment order. |

**Options:**

```go
type OneHotOptions struct {
    Columns        []string
    DropFirst      bool
    HandleNaN      NaNPolicy
    Unknown        UnknownPolicy
    Prefix         string
    Separator      string
    KeepOriginal   bool
    SortCategories bool
}

type LabelEncodeOptions struct {
    Column       string
    NewColumn    string
    SortBy       LabelSort
    HandleNaN    NaNPolicy
    Unknown      UnknownPolicy
    KeepOriginal bool
}

type OrdinalEncodeOptions struct {
    Column       string
    Order        []any
    NewColumn    string
    HandleNaN    NaNPolicy
    Unknown      UnknownPolicy
    KeepOriginal bool
}
```

`OneHotEncode` emits one `0/1` `int` column per category, named `<prefix><separator><category>`; by default the prefix is the source column name and the separator is `"_"`. `DropFirst` omits the first category as the reference level. `LabelEncode` maps each class to an integer id. `OrdinalEncode` uses the explicit `Order` slice as `0..n-1`.

**Encoder interface and introspection:**

```go
type Encoder interface {
    Transform(dt *DataTable) (*DataTable, error)
    InverseTransform(dt *DataTable) (*DataTable, error)
    Kind() string
}

func (e *OneHotEncoder) Categories() map[string][]any
func (e *OneHotEncoder) OutputColumns() []string
func (e *LabelEncoder) Classes() []any
func (e *LabelEncoder) Inverse(values ...any) ([]any, error)
func (e *OrdinalEncoder) Classes() []any
func (e *OrdinalEncoder) Inverse(values ...any) ([]any, error)
```

**Example — fit, transform, inverse:**

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
)

func main() {
    train := insyra.NewDataTable(
        insyra.NewDataList("red", "blue", "red").SetName("color"),
        insyra.NewDataList(10, 20, 30).SetName("value"),
    )

    encoded, enc, err := train.OneHotEncode(insyra.OneHotOptions{
        Columns:   []string{"color"},
        DropFirst: true,
        Unknown:   insyra.UnknownIgnore,
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(encoded.ColNames()) // [color_blue value]

    test := insyra.NewDataTable(
        insyra.NewDataList("blue", "green").SetName("color"),
        insyra.NewDataList(40, 50).SetName("value"),
    )
    encodedTest, err := enc.Transform(test)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(encodedTest.GetColByName("color_blue").Data()) // [1 0]

    originalShape, err := enc.InverseTransform(encoded)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(originalShape.ColNames()) // [color value]
}
```

**Example — categorical predictor into `stats.LinearRegression`:**

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/stats"
)

func main() {
    dt := insyra.NewDataTable(
        insyra.NewDataList(100.0, 130.0, 110.0, 150.0).SetName("sales"),
        insyra.NewDataList("basic", "pro", "basic", "pro").SetName("plan"),
    )

    x, _, err := dt.OneHotEncode(insyra.OneHotOptions{
        Columns:   []string{"plan"},
        DropFirst: true, // baseline = first-seen category ("basic")
    })
    if err != nil {
        log.Fatal(err)
    }

    fit, err := stats.LinearRegression(
        dt.GetColByName("sales"),
        x.GetColByName("plan_pro"),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(fit.Coefficients) // intercept, effect of plan_pro
}
```

### Feature Scaling

```go
func (dt *DataTable) StandardScale(cols ...string) (*DataTable, *StandardScaler, error)
func (dt *DataTable) MinMaxScale(featureMin, featureMax float64, cols ...string) (*DataTable, *MinMaxScaler, error)
func (dt *DataTable) RobustScale(cols ...string) (*DataTable, *RobustScaler, error)
func (dt *DataTable) MaxAbsScale(cols ...string) (*DataTable, *MaxAbsScaler, error)
```

**Description:** Feature scalers fit numeric scaling parameters once and reuse them. Each method returns a fresh `*DataTable` (the receiver is not modified) plus a fitted scaler. The scaler can `Transform` and `InverseTransform` other tables with the same parameters.

**Why this is not `DataList.Normalize` / `Standardize`:** `Normalize()` and `Standardize()` are stateless and modify the list in place — they recompute min/max or mean/std from whatever data they are handed. That is exactly what you must *not* do to a test set: scaling test data with its own statistics leaks information and makes train/test results incomparable. A `Scaler` fits on the **training** set and applies those frozen parameters to the test set, which is the correct, leakage-free workflow.

**Choosing a scaler:**

| Scaler | Centers on | Scales by | Use when |
|---|---|---|---|
| `StandardScaler` | mean | sample std dev (matches `Standardize`) | roughly Gaussian features; the common default |
| `MinMaxScaler` | min | range, into `[featureMin, featureMax]` | you need a bounded range (e.g. `[0,1]`) and have few outliers |
| `RobustScaler` | median | IQR (Q3−Q1) | features have outliers that would distort mean/std |
| `MaxAbsScaler` | 0 | max absolute value, into `[-1,1]` | sparse or sign-meaningful data you don't want to shift |

**Behavior:**

- Column references resolve by name first, then Excel-style index (`"A"`, `"B"`, ...). `cols` is required.
- Only the listed columns are scaled; other columns pass through unchanged, preserving column order, names, table name, and row names.
- `nil` and `NaN` are preserved and excluded from fitting (they do not affect the computed parameters).
- A non-numeric, non-missing value in a target column is an error.
- `Transform` errors if a fitted column is missing from the input. `InverseTransform` only restores fitted columns that are present and passes others through (so a prediction table covering a subset still works).
- Constant / degenerate columns (`std==0`, `max==min`, `IQR==0`, `maxAbs==0`) do not panic: the fitted training data maps to the degenerate output (`0`, or `featureMin` for min-max) and the inverse round-trips.

**Interface and introspection:**

```go
type Scaler interface {
    Fit(dt *DataTable, cols ...string) error
    Transform(dt *DataTable) (*DataTable, error)
    FitTransform(dt *DataTable, cols ...string) (*DataTable, error)
    InverseTransform(dt *DataTable) (*DataTable, error)
    Params() map[string]ScalerParams
    Kind() string
}

func NewStandardScaler() *StandardScaler
func NewMinMaxScaler(featureMin, featureMax float64) *MinMaxScaler
func NewDefaultMinMaxScaler() *MinMaxScaler // [0, 1]
func NewRobustScaler() *RobustScaler
func NewMaxAbsScaler() *MaxAbsScaler
```

`Params()` returns the fitted parameters keyed by column name (`Mean`/`Std`, `Min`/`Max`/`OutputMin`/`OutputMax`, `Median`/`Q1`/`Q3`/`IQR`, or `MaxAbs` depending on the kind).

There is also a DataList-oriented counterpart (`DataListScaler`) on every scaler: `FitDataList`, `TransformDataList`, `FitTransformDataList`, `InverseTransformDataList`, each returning a new `*DataList`.

**Example — fit on train, transform test (no leakage):**

```go
package main

import (
    "fmt"
    "log"

    "github.com/HazelnutParadise/insyra"
)

func main() {
    data := insyra.NewDataTable(
        insyra.NewDataList(20.0, 30.0, 40.0, 50.0, 60.0).SetName("Age"),
        insyra.NewDataList(30.0, 45.0, 50.0, 70.0, 90.0).SetName("Income"),
    )
    train, test := data.TrainTestSplit(0.8, insyra.SamplingOptions{UseSeed: true, Seed: 42})

    // Fit the scaler on the training set only.
    sc := insyra.NewStandardScaler()
    trainScaled, err := sc.FitTransform(train, "Age", "Income")
    if err != nil {
        log.Fatal(err)
    }

    // Apply the SAME parameters to the test set.
    testScaled, err := sc.Transform(test)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(sc.Params()["Age"]) // {Age standard <mean> <std> ...}
    _ = trainScaled
    _ = testScaled

    // Recover the original scale (e.g. for predictions).
    restored, err := sc.InverseTransform(trainScaled)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(restored.GetColByName("Age").Data())
}
```

### Window / sequence transforms (Shift / Diff / PctChange / Cum\* / Rolling / Expanding)

```go
func (dt *DataTable) ShiftCol(col string, periods int, fill ...any) *DataList
func (dt *DataTable) DiffCol(col string, periods int) *DataList
func (dt *DataTable) PctChangeCol(col string, periods int) *DataList
func (dt *DataTable) CumSumCol(col string) *DataList
func (dt *DataTable) CumProdCol(col string) *DataList
func (dt *DataTable) CumMaxCol(col string) *DataList
func (dt *DataTable) CumMinCol(col string) *DataList
func (dt *DataTable) RollingCol(col string, opts RollingOptions) *RollingDataList
func (dt *DataTable) ExpandingCol(col string, minObs int) *ExpandingDataList
```

**Description:** Per-column time-series / sequence transforms. Each method resolves `col` by **name first**, then by Excel-style index (`"A"`, `"B"`, ...), and runs the matching operation on a snapshot of that column. The returned `*DataList` (or builder) has the same length as the source so the result lines up with neighbouring columns when appended back.

The scalar transforms (`ShiftCol` / `DiffCol` / `PctChangeCol` / `Cum*Col`) return `*DataList` directly. `RollingCol` and `ExpandingCol` return builders — pick a reducer (`.Mean()` / `.Sum()` / `.Min()` / `.Max()` / `.Median()` / `.Std()` / `.Var()` / `.Apply(...)` / `.Corr(...)`) to materialise the column. See [DataList.Rolling](DataList.md#rolling) and [DataList.Expanding](DataList.md#expanding) for the full reducer list and `RollingOptions` semantics.

**Missing-value behaviour:** edge positions (e.g. the first row of `ShiftCol(_, 1)` or partial windows below `MinObs`) emit `nil`. `Shift` works on any column type (including strings / bools); the rest coerce numerically and emit `nil` for cells that aren't numeric.

**Errors:** When `col` cannot be resolved, the method records a warning on `dt.Err()` and returns an empty `*DataList`.

**Examples:**

```go
// Date | Price
dt := insyra.NewDataTable(/* date, price */)

prev   := dt.ShiftCol("price", 1)                                            // lag-1
ret    := dt.PctChangeCol("price", 1)                                        // simple return
cum    := dt.CumSumCol("price")                                              // running total
hwm    := dt.CumMaxCol("price")                                              // historical high
ma7    := dt.RollingCol("price", insyra.RollingOptions{Window: 7}).Mean()    // 7-day MA
ewmean := dt.ExpandingCol("price", 1).Mean()                                 // expanding mean

// Attach results back to the table.
ma7.SetName("ma7")
dt.AppendCols(ma7)
```

#### GroupBy-aware versions

```go
func (g *GroupedDataTable) ShiftCol(col string, periods int, fill ...any) *GroupedColumnTransform
func (g *GroupedDataTable) DiffCol(col string, periods int) *GroupedColumnTransform
func (g *GroupedDataTable) PctChangeCol(col string, periods int) *GroupedColumnTransform
func (g *GroupedDataTable) CumSumCol(col string) *GroupedColumnTransform
func (g *GroupedDataTable) CumProdCol(col string) *GroupedColumnTransform
func (g *GroupedDataTable) CumMaxCol(col string) *GroupedColumnTransform
func (g *GroupedDataTable) CumMinCol(col string) *GroupedColumnTransform
func (g *GroupedDataTable) RollingCol(col string, opts RollingOptions) *GroupedRollingCol
func (g *GroupedDataTable) ExpandingCol(col string, minObs int) *GroupedExpandingCol

func (t *GroupedColumnTransform) As(name string) *DataList
```

**Description:** Same operations as above, but scoped to each group independently. Each method returns a builder whose terminal call `.As(name)` materialises a single `*DataList` aligned to the **parent's original row order**. Internally, each group is transformed in isolation and the results are scattered back to the rows that contributed to that group, so the output sits naturally beside the source column.

For `RollingCol` / `ExpandingCol`, the builder exposes the same reducers as the ungrouped form, each of which returns a `*GroupedColumnTransform` ready for `.As`.

**Example — panel data lag/rolling per id:**

```go
//   id | t | price
//   A  | 1 | 10
//   B  | 1 | 100
//   A  | 2 | 12
//   B  | 2 | 110
//   A  | 3 | 11
//   B  | 3 | 105
prev := dt.GroupBy("id").ShiftCol("price", 1).As("prev_price")
// Per group, lag-1: A -> [nil, 10, 12]; B -> [nil, 100, 110]
// Aligned to original row order: [nil, nil, 10, 100, 12, 110]

ma3 := dt.GroupBy("id").RollingCol("price", insyra.RollingOptions{Window: 3}).Mean().As("ma3")
cum := dt.GroupBy("id").CumSumCol("price").As("cum_price")

dt.AppendCols(prev, ma3, cum)
```

**Cross-language validation:** all algorithms (Shift, Diff, PctChange, Cum*, Rolling, Expanding) are validated against pandas/numpy via fixtures committed in `testdata/window_fixtures.json`. Refresh them with `python testdata/gen_window_fixtures.py` when adding cases.

### AppendCols

```go
func (dt *DataTable) AppendCols(columns ...*DataList) *DataTable
```

**Description:** Appends columns to the DataTable.

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

```go
func (dt *DataTable) AppendRowsByColIndex(rowsData ...map[string]any) *DataTable
```

**Description:** Appends rows using column indices.

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

```go
func (dt *DataTable) AppendRowsByColName(rowsData ...map[string]any) *DataTable
```

**Description:** Appends rows using column names.

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

```go
func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) *DataTable
```

**Description:** Appends rows to the DataTable, with each row represented by a DataList.

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

```go
func (dt *DataTable) GetElement(rowIndex int, columnIndex string) any
```

**Description:** Gets the value at a specific row and column.

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

```go
func (dt *DataTable) GetCol(index string) *DataList
```

**Description:** Gets a column by its index.

**Parameters:**

- `index`: Column index (A, B, C...)

**Returns:**

- `*DataList`: The specified column

**Example:**

```go
column := dt.GetCol("A")
```

### GetColByNumber

```go
func (dt *DataTable) GetColByNumber(index int) *DataList
```

**Description:** Gets a column by its numeric index.

**Parameters:**

- `index`: Numeric column index (0-based)

**Returns:**

- `*DataList`: The specified column

**Example:**

```go
column := dt.GetColByNumber(0)
```

### GetColByName

```go
func (dt *DataTable) GetColByName(name string) *DataList
```

**Description:** Gets a column by its name.

**Parameters:**

- `name`: The name of the column.

**Returns:**

- `*DataList`: The specified column.

**Example:**

```go
column := dt.GetColByName("column_name")
```

### GetRow

```go
func (dt *DataTable) GetRow(index int) *DataList
```

**Description:** Gets a row by its index.

**Parameters:**

- `index`: Row index (0-based)

**Returns:**

- `*DataList`: The specified row

**Example:**

```go
row := dt.GetRow(0)
```

### GetRowByName

```go
func (dt *DataTable) GetRowByName(name string) *DataList
```

**Description:** Gets a row by its name.

**Parameters:**

- `name`: The name of the row.

**Returns:**

- `*DataList`: The specified row. Returns nil if the row name is not found.

**Example:**

```go
row := dt.GetRowByName("row_name")
```

### UpdateElement

```go
func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value any) *DataTable
```

**Description:** Updates the value at a specific position. Returns the table to support chaining calls.

**Parameters:**

- `rowIndex`: Row index (0-based)
- `columnIndex`: Column index (A, B, C...)
- `value`: New value

**Returns:**

- `*DataTable`: Return value.

**Example:**

```go
// single call
dt.UpdateElement(0, "A", "Jane")

// chaining example
newCol := insyra.NewDataList(1, 2, 3, 4)
dt.UpdateElement(0, "A", "Jane").UpdateCol("B", newCol)
```

### UpdateCol

```go
func (dt *DataTable) UpdateCol(index string, dl *DataList) *DataTable
```

**Description:** Updates an entire column with new data. Returns the table to support chaining calls.

**Parameters:**

- `index`: Column index (A, B, C...)
- `dl`: New DataList to replace the column

**Returns:**

- `*DataTable`: Return value.

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

```go
func (dt *DataTable) UpdateColByNumber(index int, dl *DataList) *DataTable
```

**Description:** Updates an entire column by its numeric index. Returns the table to support chaining calls.

**Parameters:**

- `index`: Numeric column index (0-based)
- `dl`: New DataList to replace the column

**Returns:**

- `*DataTable`: Return value.

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

```go
func (dt *DataTable) UpdateRow(index int, dl *DataList) *DataTable
```

**Description:** Updates an entire row with new data. Returns the table to support chaining calls.

**Parameters:**

- `index`: Row index (0-based)
- `dl`: New DataList to replace the row

**Returns:**

- `*DataTable`: Return value.

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

```go
func (dt *DataTable) GetElementByNumberIndex(rowIndex int, columnIndex int) any
```

**Description:** Gets the value at a specific row and column using numeric indices.

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

```go
func (dt *DataTable) SetColToRowNames(columnIndex string) *DataTable
```

**Description:** Sets the row names to the values of the specified column and drops the column.

**Parameters:**

- `columnIndex`: Column index (A, B, C...) to use as row names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetColToRowNames("A") // Use column A values as row names
```

### SetRowToColNames

```go
func (dt *DataTable) SetRowToColNames(rowIndex int) *DataTable
```

**Description:** Sets the column names to the values of the specified row and drops the row.

**Parameters:**

- `rowIndex`: Row index (0-based) to use as column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetRowToColNames(0) // Use first row values as column names
```

### ChangeRowName

```go
func (dt *DataTable) ChangeRowName(oldName, newName string) *DataTable
```

**Description:** Changes the name of a row.

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

```go
func (dt *DataTable) SetRowNameByIndex(index int, name string)
```

**Description:** Sets the name of a row by its index.

**Parameters:**

- `index`: Row index (0-based)
- `name`: New row name

**Returns:**

- None.

**Example:**

```go
dt.SetRowNameByIndex(0, "FirstRow") // Set the name of the first row to "FirstRow"
```

### GetRowNameByIndex

```go
func (dt *DataTable) GetRowNameByIndex(index int) (string, bool)
```

**Description:** Gets the name of a row by its index.

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

```go
func (dt *DataTable) GetRowIndexByName(name string) (int, bool)
```

**Description:** Gets the index of a row by its name. This is the inverse lookup of `GetRowNameByIndex`.

**Parameters:**

- `name`: The name of the row to find.

**Returns:**

- `int`: The row index (0-based). Returns `-1` if the row name does not exist.
- `bool`: `true` if the row name exists, `false` otherwise.

> [!NOTE]
> Since Insyra's Get methods usually support -1 as an index (representing the last element), always check the boolean return value to distinguish between "name not found" and "last row".

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

```go
func (dt *DataTable) RowNamesToFirstCol() *DataTable
```

**Description:** Moves all row names to the first column of the DataTable and clears the row names mapping.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// If DataTable has rows named "row1", "row2", "row3"
// After calling RowNamesToFirstCol(), these names will appear in the first column
dt.RowNamesToFirstCol()
```

### DropRowNames

```go
func (dt *DataTable) DropRowNames() *DataTable
```

**Description:** Removes all row names from the DataTable.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
// Remove all row names
dt.DropRowNames()
```

### RowNames

```go
func (dt *DataTable) RowNames() []string
```

**Description:** Returns a slice containing all row names in order. Returns empty strings for rows without names.

**Parameters:**

- None.

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

```go
func (dt *DataTable) SetRowNames(rowNames []string) *DataTable
```

**Description:** Sets the row names of the DataTable using a slice of strings. Only sets names for existing rows; excess names are ignored.

**Parameters:**

- `rowNames`: Slice of strings representing the new row names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetRowNames([]string{"Row1", "Row2", "Row3"}) // Set row names
```

### ChangeColName

```go
func (dt *DataTable) ChangeColName(oldName, newName string) *DataTable
```

**Description:** Changes the name of a column.

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

```go
func (dt *DataTable) GetColNameByNumber(index int) string
```

**Description:** Gets the name of a column by its numeric index.

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

```go
func (dt *DataTable) GetColNameByIndex(index string) string
```

**Description:** Gets the name of a column by its Excel-style index (A, B, C, ..., Z, AA, AB, ...).

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

```go
func (dt *DataTable) GetColNumberByName(name string) int
```

**Description:** Gets the numeric index of a column by its name.

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

```go
func (dt *DataTable) GetColIndexByName(name string) string
```

**Description:** Gets the column index (A, B, C, ...) by its name.

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

```go
func (dt *DataTable) GetColIndexByNumber(number int) string
```

**Description:** Gets the column index (A, B, C, ...) by its numeric index (0, 1, 2, ...).

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

```go
func (dt *DataTable) SetColNameByNumber(numberIndex int, name string) *DataTable
```

**Description:** Sets the name of a column by its numeric index.

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

```go
func (dt *DataTable) SetColNameByIndex(index string, name string) *DataTable
```

**Description:** Sets the name of a column by its alphabetical index.

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

```go
func (dt *DataTable) SetColNames(colNames []string) *DataTable
```

**Description:** Sets the column names of the DataTable using a slice of strings. If the slice has more elements than existing columns, new columns will be added. If the slice has fewer elements, excess columns will be set to empty names.

**Parameters:**

- `colNames`: Slice of strings representing the new column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetColNames([]string{"Name", "Age", "Role"}) // Set column names
```

### SetHeaders

```go
func (dt *DataTable) SetHeaders(headers []string) *DataTable
```

**Description:** Alias for SetColNames, sets the column names of the DataTable.

**Parameters:**

- `headers`: Slice of strings representing the new column names

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.SetHeaders([]string{"Name", "Age", "Role"}) // Set column names
```

### ColNamesToFirstRow

```go
func (dt *DataTable) ColNamesToFirstRow() *DataTable
```

**Description:** Moves all column names to the first row of data and then clears the column names.

**Parameters:**

- None.

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

```go
func (dt *DataTable) DropColNames() *DataTable
```

**Description:** Removes all column names, setting them to empty strings.

**Parameters:**

- None.

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

```go
func (dt *DataTable) ColNames() []string
```

**Description:** Returns a slice containing all column names in order.

**Parameters:**

- None.

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

```go
func (dt *DataTable) Headers() []string
```

**Description:** Alias for ColNames, returns a slice containing all column names in order.

**Parameters:**

- None.

**Returns:**

- `[]string`: Slice of column names

**Example:**

```go
headers := dt.Headers() // Same as dt.ColNames()
```

### DropColsByName

```go
func (dt *DataTable) DropColsByName(columnNames ...string) *DataTable
```

**Description:** Drops columns by their names.

**Parameters:**

- `columnNames`: Names of columns to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsByName("Age", "Address")
```

### DropRowsByIndex

```go
func (dt *DataTable) DropRowsByIndex(rowIndices ...int) *DataTable
```

**Description:** Drops rows by their numeric indices (0-based).

**Parameters:**

- `rowIndices`: Numeric row indices (0-based) to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsByIndex(0, 2, 5) // Drops rows at indices 0, 2, and 5
```

### DropRowsByName

```go
func (dt *DataTable) DropRowsByName(rowNames ...string) *DataTable
```

**Description:** Drops rows by their names.

**Parameters:**

- `rowNames`: Names of rows to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsByName("row1", "row2")
```

### DropColsByIndex

```go
func (dt *DataTable) DropColsByIndex(columnIndices ...string) *DataTable
```

**Description:** Drops columns by their letter indices (e.g., "A", "B", "C").

**Parameters:**

- `columnIndices`: Column letter indices (A, B, C...) to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsByIndex("A", "C") // Drops columns A and C
```

### DropColsByNumber

```go
func (dt *DataTable) DropColsByNumber(columnIndices ...int) *DataTable
```

**Description:** Drops columns by their numeric indices (0-based).

**Parameters:**

- `columnIndices`: Numeric column indices (0-based) to drop

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsByNumber(0, 2) // Drops first and third columns (columns at index 0 and 2)
```

### DropColsContainString

```go
func (dt *DataTable) DropColsContainString() *DataTable
```

**Description:** Drops columns that contain any string elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainString() // Drops all columns that have at least one string element
```

### DropColsContainNumber

```go
func (dt *DataTable) DropColsContainNumber() *DataTable
```

**Description:** Drops columns that contain any numeric elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNumber() // Drops all columns that have at least one numeric element
```

### DropColsContainNil

```go
func (dt *DataTable) DropColsContainNil() *DataTable
```

**Description:** Drops columns that contain any nil (null) elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNil() // Drops all columns that have at least one nil element
```

### DropColsContainNaN

```go
func (dt *DataTable) DropColsContainNaN() *DataTable
```

**Description:** Drops columns that contain any NaN (Not a Number) elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropColsContainNaN() // Drops all columns that have at least one NaN element
```

### DropColsContain

```go
func (dt *DataTable) DropColsContain(value ...any) *DataTable
```

**Description:** Drops columns that contain the specified value(s).

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

```go
func (dt *DataTable) DropColsContainExcelNA() *DataTable
```

**Description:** Drops columns that contain Excel NA values ("#N/A").

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Drop columns containing Excel NA values
dt.DropColsContainExcelNA()
```

### DropRowsContainString

```go
func (dt *DataTable) DropRowsContainString() *DataTable
```

**Description:** Drops rows that contain any string elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainString() // Drops all rows that have at least one string element
```

### DropRowsContainNumber

```go
func (dt *DataTable) DropRowsContainNumber() *DataTable
```

**Description:** Drops rows that contain any numeric elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNumber() // Drops all rows that have at least one numeric element
```

### DropRowsContainNil

```go
func (dt *DataTable) DropRowsContainNil() *DataTable
```

**Description:** Drops rows that contain any nil (null) elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNil() // Drops all rows that have at least one nil element
```

### DropRowsContainNaN

```go
func (dt *DataTable) DropRowsContainNaN() *DataTable
```

**Description:** Drops rows that contain any NaN (Not a Number) elements.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
dt.DropRowsContainNaN() // Drops all rows that have at least one NaN element
```

### DropRowsContain

```go
func (dt *DataTable) DropRowsContain(value ...any) *DataTable
```

**Description:** Drops rows that contain the specified value(s).

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

```go
func (dt *DataTable) DropRowsContainExcelNA() *DataTable
```

**Description:** Drops rows that contain Excel NA values ("#N/A").

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The modified DataTable

**Example:**

```go
// Drop rows containing Excel NA values
dt.DropRowsContainExcelNA()
```

### SwapColsByName

```go
func (dt *DataTable) SwapColsByName(columnName1 string, columnName2 string) *DataTable
```

**Description:** Swaps two columns by their names.

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

```go
func (dt *DataTable) SwapColsByIndex(columnIndex1 string, columnIndex2 string) *DataTable
```

**Description:** Swaps two columns by their letter indices (e.g., "A", "B").

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

```go
func (dt *DataTable) SwapColsByNumber(columnNumber1 int, columnNumber2 int) *DataTable
```

**Description:** Swaps two columns by their numerical indices (0-based).

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

```go
func (dt *DataTable) SwapRowsByIndex(rowIndex1 int, rowIndex2 int) *DataTable
```

**Description:** Swaps two rows by their numerical indices (0-based).

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

```go
func (dt *DataTable) SwapRowsByName(rowName1 string, rowName2 string) *DataTable
```

**Description:** Swaps two rows by their names.

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

```go
func (dt *DataTable) Count() int
```

**Description:** Returns the number of rows in the DataTable.

**Parameters:**

- None.

**Returns:**

- `int`: Number of rows

**Example:**

```go
count := dt.Count()
fmt.Printf("Table has %d rows\n", count)
```

### Counter

```go
func (dt *DataTable) Counter() map[any]int
```

**Description:** Returns the number of occurrences of each value in the DataTable.

**Parameters:**

- None.

**Returns:**

- `map[any]int`: Map containing the count of each value in the DataTable

**Example:**

```go
counts := dt.Counter()
fmt.Printf("Value counts: %v\n", counts)
```

### GetCreationTimestamp

```go
func (dt *DataTable) GetCreationTimestamp() int64
```

**Description:** Gets the creation timestamp of the DataTable.

**Parameters:**

- None.

**Returns:**

- `int64`: Unix timestamp when the table was created

**Example:**

```go
timestamp := dt.GetCreationTimestamp()
fmt.Printf("Created at: %d\n", timestamp)
```

### GetLastModifiedTimestamp

```go
func (dt *DataTable) GetLastModifiedTimestamp() int64
```

**Description:** Gets the last modified timestamp of the DataTable.

**Parameters:**

- None.

**Returns:**

- `int64`: Unix timestamp when the table was last modified

**Example:**

```go
timestamp := dt.GetLastModifiedTimestamp()
fmt.Printf("Last modified at: %d\n", timestamp)
```

### SimpleRandomSample

```go
func (dt *DataTable) SimpleRandomSample(sampleSize int) *DataTable
```

**Description:** Performs simple random sampling on the DataTable.

> **Deprecated:** Use [`Sample(n, false)`](#sample--samplefrac--shuffle--traintestsplit) instead, which shares the `SamplingOptions` (seed/reproducibility) surface with the other sampling methods. Note `Sample` records an error and returns an empty table when `n` exceeds the row count, rather than cloning the whole table.

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

### Sample / SampleFrac / Shuffle / TrainTestSplit

```go
func (dt *DataTable) Sample(n int, withReplacement bool, options ...SamplingOptions) *DataTable
func (dt *DataTable) SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataTable
func (dt *DataTable) Shuffle(options ...SamplingOptions) *DataTable
func (dt *DataTable) TrainTestSplit(trainFrac float64, options ...SamplingOptions) (*DataTable, *DataTable)
```

**Description:** Performs row-wise random sampling, shuffling, and train/test splitting. DataTable operations always move whole rows together, so every column and row name remains aligned.

**Options:**

```go
type SamplingOptions struct {
    Seed          uint64
    UseSeed       bool
    PreserveOrder bool // TrainTestSplit only; false means shuffle before split.
}
```

Use `SamplingOptions{UseSeed: true, Seed: 42}` for reproducible samples. `TrainTestSplit` shuffles before splitting by default; set `PreserveOrder: true` for ordered splits, such as time-series data.

**Examples:**

```go
sample := dt.Sample(100, false, insyra.SamplingOptions{UseSeed: true, Seed: 42})
preview := dt.SampleFrac(0.05, false)
shuffled := dt.Shuffle()

train, test := dt.TrainTestSplit(0.8, insyra.SamplingOptions{UseSeed: true, Seed: 42})
orderedTrain, orderedTest := dt.TrainTestSplit(0.8, insyra.SamplingOptions{PreserveOrder: true})
```

**Notes:**

- `SampleFrac` and `TrainTestSplit` use `floor(frac * rows)`, with a minimum of 1 row for non-empty data.
- `TrainTestSplit` requires `trainFrac` in the open interval `(0, 1)` and a row count large enough that neither split is empty; otherwise it records an error via `dt.Err()` and returns two empty DataTables.
- Without replacement, `n > rows` records an error and returns an empty DataTable.
- With replacement, duplicate source rows may appear in the output.
- Invalid fractions, empty input, and invalid sample sizes are recorded via `dt.Err()`.

## Data Replacement

DataTable provides several methods to replace values within the entire table, a specific row, or a specific column.

### Missing-Value Fill Methods

```go
func (dt *DataTable) FillForward(limit int, cols ...string) *DataTable
func (dt *DataTable) FillBackward(limit int, cols ...string) *DataTable
func (dt *DataTable) FillWithMean(cols ...string) *DataTable
func (dt *DataTable) FillWithMedian(cols ...string) *DataTable
func (dt *DataTable) FillWithMode(cols ...string) *DataTable
func (dt *DataTable) FillByInterpolation(cols ...string) *DataTable
```

**Description:** Fills `nil` and `math.NaN()` values column by column. When `cols` is omitted, all applicable columns are processed. Mean, median, and interpolation apply only to numeric columns; mode and forward/backward fill can apply to any selected column.

**Parameters:**

- `limit`: Maximum consecutive values to fill for forward/backward fill. `0` means unlimited.
- `cols` (optional): Column names or Excel-style indices to process.

**Returns:**

- `*DataTable`: Reference to the modified DataTable.

**Example:**

```go
dt.FillWithMedian("revenue", "cost")
dt.FillForward(2, "status")
dt.FillByInterpolation() // all numeric columns
```

### Replace

```go
func (dt *DataTable) Replace(oldValue, newValue any) *DataTable
```

**Description:** Replaces all occurrences of `oldValue` with `newValue` in the entire DataTable.

**Parameters:**

- `oldValue`: Input value for `oldValue`. Type: `any`.
- `newValue`: Input value for `newValue`. Type: `any`.

**Returns:**

- `*DataTable`: Return value.

### ReplaceNaNsWith

```go
func (dt *DataTable) ReplaceNaNsWith(newValue any) *DataTable
```

**Description:** Replaces all occurrences of `NaN` with `newValue` in the entire DataTable.

**Parameters:**

- `newValue`: Input value for `newValue`. Type: `any`.

**Returns:**

- `*DataTable`: Return value.

### ReplaceNilsWith

```go
func (dt *DataTable) ReplaceNilsWith(newValue any) *DataTable
```

**Description:** Replaces all occurrences of `nil` with `newValue` in the entire DataTable.

**Parameters:**

- `newValue`: Input value for `newValue`. Type: `any`.

**Returns:**

- `*DataTable`: Return value.

### ReplaceNaNsAndNilsWith

```go
func (dt *DataTable) ReplaceNaNsAndNilsWith(newValue any) *DataTable
```

**Description:** Replaces all occurrences of both `NaN` and `nil` with `newValue` in the entire DataTable.

**Parameters:**

- `newValue`: Input value for `newValue`. Type: `any`.

**Returns:**

- `*DataTable`: Return value.

### ReplaceInRow

```go
func (dt *DataTable) ReplaceInRow(rowIndex int, oldValue, newValue any, mode ...int) *DataTable
```

**Description:** Replaces occurrences of `oldValue` with `newValue` in a specific row.

**Parameters:**

- `rowIndex`: The index of the row.
- `oldValue`: The value to be replaced.
- `newValue`: The value to replace with.
- `mode` (optional):
  - `0` (default): Replace all occurrences.
  - `1`: Replace only the first occurrence.
  - `-1`: Replace only the last occurrence.

**Returns:**

- `*DataTable`: Return value.

### ReplaceNaNsInRow / ReplaceNilsInRow / ReplaceNaNsAndNilsInRow

Similar to `ReplaceInRow`, but specifically for `NaN`, `nil`, or both.

```go
func (dt *DataTable) ReplaceNaNsInRow(rowIndex int, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNaNsAndNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable
```

### ReplaceInCol

```go
func (dt *DataTable) ReplaceInCol(colIndex string, oldValue, newValue any, mode ...int) *DataTable
```

**Description:** Replaces occurrences of `oldValue` with `newValue` in a specific column.

**Parameters:**

- `colIndex`: The index or name of the column.
- `oldValue`: The value to be replaced.
- `newValue`: The value to replace with.
- `mode` (optional):
  - `0` (default): Replace all occurrences.
  - `1`: Replace only the first occurrence.
  - `-1`: Replace only the last occurrence.

**Returns:**

- `*DataTable`: Return value.

### ReplaceNaNsInCol / ReplaceNilsInCol / ReplaceNaNsAndNilsInCol

Similar to `ReplaceInCol`, but specifically for `NaN`, `nil`, or both.

```go
func (dt *DataTable) ReplaceNaNsInCol(colIndex string, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNilsInCol(colIndex string, newValue any, mode ...int) *DataTable
func (dt *DataTable) ReplaceNaNsAndNilsInCol(colIndex string, newValue any, mode ...int) *DataTable
```

## Column Calculation

### AddColUsingCCL

```go
func (dt *DataTable) AddColUsingCCL(newColName, ccl string) *DataTable
```

**Description:** Adds a new column to the DataTable by evaluating a Column Calculation Language (CCL) expression on each row.

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

```go
func (dt *DataTable) ExecuteCCL(ccl string) *DataTable
```

**Description:** Executes one or more CCL statements on the DataTable. Statements are separated by newlines and executed sequentially. Each statement sees the results of previous statements.

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

```go
func (dt *DataTable) FindRowsIfContains(value any) []int
```

**Description:** Returns the indices of rows that contain the given element.

**Parameters:**

- `value`: Value to search for

**Returns:**

- `[]int`: Slice of row indices that contain the value

**Example:**

```go
rowIndices := dt.FindRowsIfContains("John")
```

### FindRowsIfContainsAll

```go
func (dt *DataTable) FindRowsIfContainsAll(values ...any) []int
```

**Description:** Returns the indices of rows that contain all the given elements.

**Parameters:**

- `values`: Values to search for

**Returns:**

- `[]int`: Slice of row indices that contain all values

**Example:**

```go
rowIndices := dt.FindRowsIfContainsAll("John", 25)
```

### FindColsIfContains

```go
func (dt *DataTable) FindColsIfContains(value any) []string
```

**Description:** Returns the indices of columns that contain the given element.

**Parameters:**

- `value`: Value to search for

**Returns:**

- `[]string`: Slice of column indices that contain the value

**Example:**

```go
colIndices := dt.FindColsIfContains("Engineer")
```

### FindColsIfContainsAll

```go
func (dt *DataTable) FindColsIfContainsAll(values ...any) []string
```

**Description:** Returns the indices of columns that contain all the given elements.

**Parameters:**

- `values`: Values to search for

**Returns:**

- `[]string`: Slice of column indices that contain all the values

**Example:**

```go
colIndices := dt.FindColsIfContainsAll("Engineer", 25)
```

### FindRowsIfAnyElementContainsSubstring

```go
func (dt *DataTable) FindRowsIfAnyElementContainsSubstring(substring string) []int
```

**Description:** Returns the indices of rows where at least one element contains the given substring.

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]int`: Slice of row indices where at least one element contains the substring

**Example:**

```go
rowIndices := dt.FindRowsIfAnyElementContainsSubstring("John")
```

### FindRowsIfAllElementsContainSubstring

```go
func (dt *DataTable) FindRowsIfAllElementsContainSubstring(substring string) []int
```

**Description:** Returns the indices of rows where all elements contain the given substring.

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]int`: Slice of row indices where all elements contain the substring

**Example:**

```go
rowIndices := dt.FindRowsIfAllElementsContainSubstring("data")
```

### FindColsIfAnyElementContainsSubstring

```go
func (dt *DataTable) FindColsIfAnyElementContainsSubstring(substring string) []string
```

**Description:** Returns the indices of columns where at least one element contains the given substring.

**Parameters:**

- `substring`: Substring to search for

**Returns:**

- `[]string`: Slice of column indices where at least one element contains the substring

**Example:**

```go
colIndices := dt.FindColsIfAnyElementContainsSubstring("Engineer")
```

### FindColsIfAllElementsContainSubstring

```go
func (dt *DataTable) FindColsIfAllElementsContainSubstring(substring string) []string
```

**Description:** Returns the indices of columns where all elements contain the given substring.

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

```go
func (dt *DataTable) Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable
```

**Description:** Filters the DataTable using a custom filter function. Keeps only rows where the filter function returns true for at least one cell.

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

```go
func (dt *DataTable) FilterByCustomElement(f func(value any) bool) *DataTable
```

**Description:** Filters the DataTable based on a custom function applied to each element.

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

```go
func (dt *DataTable) FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable
```

**Description:** Filters rows based on a custom function that checks each cell. Keeps only rows where the filter function returns true for at least one cell.

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

```go
func (dt *DataTable) FilterCols(filterFunc func(rowIndex int, rowName string, x any) bool) *DataTable
```

**Description:** Filters columns based on a custom function applied to each cell. Keeps only columns where the filter function returns true for at least one cell in that column.

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

```go
func (dt *DataTable) FilterColsByColNameEqualTo(columnName string) *DataTable
```

**Description:** Filters columns by exact name match.

**Parameters:**

- `columnName`: Column name to match

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColNameEqualTo("age")
```

### FilterColsByColIndexGreaterThan

```go
func (dt *DataTable) FilterColsByColIndexGreaterThan(threshold string) *DataTable
```

**Description:** Filters columns by index greater than the specified threshold.

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexGreaterThan("B") // Columns C, D, E...
```

### FilterColsByColIndexGreaterThanOrEqualTo

```go
func (dt *DataTable) FilterColsByColIndexGreaterThanOrEqualTo(threshold string) *DataTable
```

**Description:** Filters columns by index greater than or equal to the specified threshold.

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexGreaterThanOrEqualTo("B") // Columns B, C, D...
```

### FilterColsByColIndexLessThan

```go
func (dt *DataTable) FilterColsByColIndexLessThan(threshold string) *DataTable
```

**Description:** Filters columns by index less than the specified threshold.

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexLessThan("C") // Columns A, B
```

### FilterColsByColIndexLessThanOrEqualTo

```go
func (dt *DataTable) FilterColsByColIndexLessThanOrEqualTo(threshold string) *DataTable
```

**Description:** Filters columns by index less than or equal to the specified threshold.

**Parameters:**

- `threshold`: Column index threshold (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexLessThanOrEqualTo("C") // Columns A, B, C
```

### FilterColsByColIndexEqualTo

```go
func (dt *DataTable) FilterColsByColIndexEqualTo(index string) *DataTable
```

**Description:** Filters columns by exact index match.

**Parameters:**

- `index`: Column index (A, B, C...)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColIndexEqualTo("B") // Only column B
```

### FilterColsByColNameContains

```go
func (dt *DataTable) FilterColsByColNameContains(substring string) *DataTable
```

**Description:** Filters columns whose name contains the specified substring.

**Parameters:**

- `substring`: Substring to search for in column names

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterColsByColNameContains("age") // Columns with "age" in name
```

### FilterRowsByRowNameEqualTo

```go
func (dt *DataTable) FilterRowsByRowNameEqualTo(name string) *DataTable
```

**Description:** Filters rows by exact name match.

**Parameters:**

- `name`: Row name to match

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowNameEqualTo("John")
```

### FilterRowsByRowNameContains

```go
func (dt *DataTable) FilterRowsByRowNameContains(substring string) *DataTable
```

**Description:** Filters rows whose name contains the specified substring.

**Parameters:**

- `substring`: Substring to search for in row names

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowNameContains("John") // Rows with "John" in name
```

### FilterRowsByRowIndexGreaterThan

```go
func (dt *DataTable) FilterRowsByRowIndexGreaterThan(threshold int) *DataTable
```

**Description:** Filters rows by index greater than the specified threshold.

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexGreaterThan(5) // Rows 6, 7, 8...
```

### FilterRowsByRowIndexGreaterThanOrEqualTo

```go
func (dt *DataTable) FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable
```

**Description:** Filters rows by index greater than or equal to the specified threshold.

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexGreaterThanOrEqualTo(5) // Rows 5, 6, 7...
```

### FilterRowsByRowIndexLessThan

```go
func (dt *DataTable) FilterRowsByRowIndexLessThan(threshold int) *DataTable
```

**Description:** Filters rows by index less than the specified threshold.

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexLessThan(5) // Rows 0, 1, 2, 3, 4
```

### FilterRowsByRowIndexLessThanOrEqualTo

```go
func (dt *DataTable) FilterRowsByRowIndexLessThanOrEqualTo(threshold int) *DataTable
```

**Description:** Filters rows by index less than or equal to the specified threshold.

**Parameters:**

- `threshold`: Row index threshold (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexLessThanOrEqualTo(5) // Rows 0, 1, 2, 3, 4, 5
```

### FilterRowsByRowIndexEqualTo

```go
func (dt *DataTable) FilterRowsByRowIndexEqualTo(index int) *DataTable
```

**Description:** Filters rows by exact index match.

**Parameters:**

- `index`: Row index (0-based)

**Returns:**

- `*DataTable`: New filtered DataTable

**Example:**

```go
filtered := dt.FilterRowsByRowIndexEqualTo(3) // Only row 3
```

## Statistical Analysis

### Describe

```go
type DescribeOptions struct {
    Percentiles []float64
    IncludeAll  bool
}

func (dt *DataTable) Describe(options ...DescribeOptions) *DataTable
```

**Description:** Returns a programmatic per-column summary table. Row names are statistics (`count`, `missing`, `unique`, `top`, `freq`, `mean`, `std`, `min`, percentiles, `max`) and columns are source columns.

By default, only columns whose non-missing values are all numeric are included. With `IncludeAll: true`, non-numeric and mixed columns are included with categorical statistics. `nil` and `NaN` count as missing. `Percentiles` uses values in `[0, 1]`; when omitted it defaults to `0.25`, `0.5`, and `0.75`.

**Example:**

```go
desc := dt.Describe(insyra.DescribeOptions{
    IncludeAll:  true,
    Percentiles: []float64{0.1, 0.5, 0.9},
})

byRegion := dt.GroupBy("region").Describe(insyra.DescribeOptions{IncludeAll: true})
```

### Summary

```go
func (dt *DataTable) Summary()
```

**Description:** Displays a comprehensive statistical summary of the DataTable.

**Parameters:**

- None.

**Returns:**

- None.

**Example:**

```go
dt.Summary() // Displays summary to console
```

### Size

```go
func (dt *DataTable) Size() (numRows int, numCols int)
```

**Description:** Returns the dimensions of the DataTable.

**Parameters:**

- None.

**Returns:**

- `numRows`: Number of rows
- `numCols`: Number of columns

**Example:**

```go
rows, cols := dt.Size()
fmt.Printf("Table has %d rows and %d columns\n", rows, cols)
```

### NumRows

```go
func (dt *DataTable) NumRows() int
```

**Description:** Returns the number of rows in the DataTable.

**Parameters:**

- None.

**Returns:**

- `int`: Number of rows

**Example:**

```go
rows := dt.NumRows()
fmt.Printf("Table has %d rows\n", rows)
```

### NumCols

```go
func (dt *DataTable) NumCols() int
```

**Description:** Returns the number of columns in the DataTable.

**Parameters:**

- None.

**Returns:**

- `int`: Number of columns

**Example:**

```go
cols := dt.NumCols()
fmt.Printf("Table has %d columns\n", cols)
```

### Mean

```go
func (dt *DataTable) Mean() any
```

**Description:** Calculates the mean of all numeric values in the DataTable.

**Parameters:**

- None.

**Returns:**

- `any`: Mean value of all numeric data

**Example:**

```go
mean := dt.Mean()
fmt.Printf("Overall mean: %v\n", mean)
```

## Utility Methods

### Show

```go
func (dt *DataTable) Show()
```

**Description:** Displays the DataTable content in the console.

**Parameters:**

- None.

**Returns:**

- None.

**Example:**

```go
dt.Show() // Display table content in console
```

### ShowRange

```go
func (dt *DataTable) ShowRange(startEnd ...any)
```

**Description:** Displays the DataTable with a specified range of rows.

**Parameters:**

- `startEnd`: Optional parameters to specify the range of rows to display:
  - No parameters: shows all rows
  - One positive value: shows first N rows (e.g., ShowRange(5) shows first 5 rows)
  - One negative value: shows last N rows (e.g., ShowRange(-5) shows last 5 rows)
  - Two values [start, end]: shows rows from index start (inclusive) to index end (exclusive)
  - If end is nil: shows rows from index start to the end of the table

**Returns:**

- None.

**Example:**

```go
dt.ShowRange()      // Show all rows
dt.ShowRange(5)     // Show first 5 rows
dt.ShowRange(-5)    // Show last 5 rows
dt.ShowRange(2, 10) // Show rows 2-9
dt.ShowRange(2, nil) // Show rows from index 2 to end
```

### ShowTypes

```go
func (dt *DataTable) ShowTypes()
```

**Description:** Displays the data types of each column.

**Parameters:**

- None.

**Returns:**

- None.

**Example:**

```go
dt.ShowTypes() // Display column type information
```

### ShowTypesRange

```go
func (dt *DataTable) ShowTypesRange(startEnd ...any)
```

**Description:** Displays the data types of columns within a specified range.

**Parameters:**

- `startEnd`: Optional parameters to specify the range of rows to display type information for

**Returns:**

- None.

**Example:**

```go
dt.ShowTypesRange(5)     // Show types for first 5 rows
dt.ShowTypesRange(-5)    // Show types for last 5 rows
dt.ShowTypesRange(2, 10) // Show types for rows 2-9
```

### GetName

```go
func (dt *DataTable) GetName() string
```

**Description:** Gets the DataTable name.

**Parameters:**

- None.

**Returns:**

- `string`: DataTable name

**Example:**

```go
name := dt.GetName()
fmt.Printf("Table name: %s\n", name)
```

### SetName

```go
func (dt *DataTable) SetName(name string) *DataTable
```

**Description:** Sets the DataTable name. Use snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces.

**Parameters:**

- `name`: New name (recommended: snake-style Pascal case)

**Returns:**

- `*DataTable`: The modified DataTable (for method chaining)

**Example:**

```go
dt.SetName("Factor_Loadings")
```

### Map

```go
func (dt *DataTable) Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable
```

**Description:** Applies a transformation function to all elements in the DataTable and returns a new DataTable with the transformed results. The function provides access to row index, column index, and element value for context-aware transformations.

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

```go
func (dt *DataTable) Transpose() *DataTable
```

**Description:** Transposes the DataTable (rows become columns and vice versa).

**Parameters:**

- None.

**Returns:**

- `*DataTable`: The transposed DataTable

**Example:**

```go
transposed := dt.Transpose()
```

### SortBy

```go
func (dt *DataTable) SortBy(configs ...DataTableSortConfig) *DataTable
```

**Description:**

- Supports sorting by column index, number, or name
- Multi-level sorting: sorts by the first config, then by subsequent configs for ties
- Uses stable sort to maintain relative order of equal elements
- At least one of ColumnIndex, ColumnNumber, or ColumnName must be specified

**Parameters:**

- `configs`: Variable number of DataTableSortConfig structs specifying sort criteria

**Returns:**

- `*DataTable`: The sorted DataTable (modified in-place)

Sorts the DataTable rows based on one or more column configurations. Supports multi-level sorting with stable sort.

**DataTableSortConfig:**

```go
type DataTableSortConfig struct {
    ColumnIndex  string // Column index (A, B, C...)
    ColumnNumber int    // Column number (0-based)
    ColumnName   string // Column name
    Descending   bool   // Sort in descending order
}
```

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

```go
func (dt *DataTable) Clone() *DataTable
```

**Description:**

The Clone method creates a complete deep copy of the DataTable, including:

- All columns (each DataList is also cloned)
- Column index mapping
- Row names mapping
- Table name
- Creation timestamp (original timestamp is preserved)
- Last modified timestamp (reset to current time)

The cloned DataTable is completely independent of the original, so modifications to one will not affect the other.

**Parameters:**

- None.

**Returns:**

- `*DataTable`: A new DataTable instance that is a deep copy of the original

Creates a deep copy of the DataTable.

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

```go
func (dt *DataTable) To2DSlice() [][]any
```

**Description:**

The To2DSlice method converts the DataTable into a standard Go 2D slice format. Each row in the DataTable becomes an inner slice in the returned 2D slice, and each column's value at that row position becomes an element in the inner slice.

- If a column is shorter than the maximum row count, `nil` values are used to fill the missing positions
- The returned slice is a deep copy of the data, so modifications to the slice won't affect the original DataTable
- This method is useful for interfacing with other Go libraries that expect 2D slice data structures

**Parameters:**

- None.

**Returns:**

- `[][]any`: A 2D slice where each inner slice represents a row and contains the column values for that row

Converts the DataTable to a 2D slice of any type.

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

## Error Handling

Insyra provides a comprehensive error handling system that supports both global error buffering and instance-level error tracking, designed to work seamlessly with method chaining.

### Error Handling Mechanisms

Insyra offers two complementary error handling approaches:

1. **Global Error Buffer**: A centralized error collection system that captures all warnings and errors across the application.
2. **Instance-Level Error Tracking**: Each DataTable (and DataList) instance maintains its own `lastError` field, allowing you to check errors after chained operations.

### Instance-Level Error Checking

After performing chained operations, you can check if any errors occurred using the `Err()` method:

```go
// Perform chained operations
dt.Replace(oldVal, newVal).ReplaceInRow(999, "a", "b").SortBy(config)

// Check for errors after the chain
if err := dt.Err(); err != nil {
    fmt.Printf("Error occurred: %s\n", err.Error())
    // Handle the error
}

// Clear the error for future operations
dt.ClearErr()
```

#### Available Methods

| Method                  | Description                                                                                     |
| ----------------------- | ----------------------------------------------------------------------------------------------- |
| `Err() *ErrorInfo`      | Returns the last error that occurred during a chained operation, or `nil` if no error occurred. |
| `ClearErr() *DataTable` | Clears the last error and returns the DataTable for continued chaining.                         |

### Global Error Buffer

The global error buffer collects all errors across the application. This is useful for monitoring and logging purposes.

#### Checking for Errors

```go
// Check if any errors exist
if insyra.HasError() {
    fmt.Println("Errors detected!")
}

// Check if any errors at or above a specific level exist
if insyra.HasErrorAboveLevel(insyra.LogLevelWarning) {
    fmt.Println("Warnings or higher severity errors detected!")
}

// Get the count of errors
count := insyra.GetErrorCount()
```

#### Retrieving Errors (Non-Destructive)

```go
// Peek at the first/last error without removing it
err := insyra.PeekError(insyra.ErrPoppingModeFIFO)
if err != nil {
    fmt.Printf("First error: %s\n", err.Error())
}

// Get all errors without removing them
allErrors := insyra.GetAllErrors()
for _, e := range allErrors {
    fmt.Printf("[%s] %s.%s: %s\n", e.Level, e.PackageName, e.FuncName, e.Message)
}

// Get errors filtered by level
warnings := insyra.GetErrorsByLevel(insyra.LogLevelWarning)

// Get errors filtered by package
dtErrors := insyra.GetErrorsByPackage("DataTable")
```

#### Retrieving and Removing Errors

```go
// Pop a single error (removes it from the buffer)
err := insyra.PopErrorInfo(insyra.ErrPoppingModeFIFO)

// Pop all errors (clears the buffer)
allErrors := insyra.PopAllErrors()

// Clear all errors
insyra.ClearErrors()
```

### ErrorInfo Structure

The `ErrorInfo` struct provides detailed error information:

```go
type ErrorInfo struct {
    Level       LogLevel    // Error severity level
    PackageName string      // Package where the error occurred
    FuncName    string      // Function where the error occurred
    Message     string      // Error message
    Timestamp   time.Time   // When the error occurred
}

// ErrorInfo implements the error interface
func (e ErrorInfo) Error() string
```

### Log Levels

```go
const (
    LogLevelDebug   LogLevel = iota  // Debug messages
    LogLevelInfo                     // Informational messages
    LogLevelWarning                  // Warning messages
    LogLevelFatal                    // Fatal errors
)
```

### Custom Error Handling Function

You can set a custom function to handle errors as they occur:

```go
insyra.Config.SetDefaultErrHandlingFunc(func(errType insyra.LogLevel, packageName, funcName, errMsg string) {
    // Custom error handling logic
    if errType >= insyra.LogLevelWarning {
        log.Printf("Custom handler: [%s] %s.%s: %s", errType, packageName, funcName, errMsg)
    }
})
```

### Best Practices

1. **Use Instance-Level Errors for Chained Operations**: When performing multiple chained operations, use `Err()` to check for errors at the end of the chain.

2. **Clear Errors After Handling**: Call `ClearErr()` after handling an error to prevent confusion in subsequent operations.

3. **Use Global Buffer for Monitoring**: The global error buffer is ideal for logging and monitoring across the application.

4. **Check Specific Error Levels**: Use `HasErrorAboveLevel()` to filter for errors that require immediate attention.

## AtomicDo

```go
func (dt *DataTable) AtomicDo(f func(*DataTable))
```

**Description:** `AtomicDo` provides safe, serialized access to a DataTable via a per-instance `sync.Mutex` plus a goroutine-id holder (`petermattis/goid`) for fast same-goroutine re-entry detection. All operations inside the function run in order and without races, allowing concurrent callers to compose multi-step updates safely with ~30 ns per-call overhead.

**Parameters:**

- `f`: Input value for `f`. Type: `func(*DataTable)`.

**Returns:**

- None.

**Behavior:**

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

## Notes

1. **Type Safety**: DataTable uses the `any` type to store data. Please ensure proper type conversion when operating on the data.

2. **Memory Management**: For large datasets, consider using streaming or batch processing to avoid memory overflow.

3. **Error Handling**: Some methods return `error` (e.g., file I/O, SQL operations). Many manipulation methods return `*DataTable` for chaining — check the specific function signatures and handle errors where applicable. Additionally, you can use the instance-level `Err()` method to check for errors after chained operations.

4. **Concurrency**: DataTable uses actor-style serialized execution via `AtomicDo` (internal command channel and goroutine) for thread-safety rather than a global mutex. Use `AtomicDo` for sequences that must observe consistent state and avoid long-blocking work inside it.

5. **Filter Operations**: Filter operations create new DataTable instances; original data is not modified.

6. **SQL Operations**: When using SQL-related functionality, ensure proper database connection configuration and permissions.

7. **Column Indexing**: Columns can be accessed by both alphabetical indices (A, B, C...) and numeric indices (0, 1, 2...).

8. **Method Chaining**: Many methods return `*DataTable` to support method chaining for fluent API usage.

9. **Naming Conventions**: Table names, column names, and row names should use snake-style Pascal case (e.g., `Factor_Loadings`) to avoid spelling errors caused by spaces.
