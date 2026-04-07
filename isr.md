# [ isr ] Package

## Overview

The `isr` package provides a simplified, method-chaining syntax for the `insyra` data analysis library.

**Package Name**: `isr` (stands for "insyra" - first letters of three syllables)

**Description:** Syntax sugar and fluent API for insyra data structures

**Key Feature**: Full method chaining support

## Quick Start

```go
import "github.com/HazelnutParadise/insyra/isr"

dl := isr.DL.Of(1, 2, 3, 4).Append(5)
dl.Show()
```

## Installation

```bash
go get github.com/HazelnutParadise/insyra/isr
```

```go
import "github.com/HazelnutParadise/insyra/isr"
```

## Core Concepts

### Pre-defined Variables

- `isr.DL` - Pre-defined DataList instance for creating new DataLists
- `isr.DT` - Pre-defined DataTable instance for creating new DataTables

### Type Aliases

- `isr.Row` - `map[any]any` representing a table row
- `isr.Rows` - `[]Row` slice of table rows
- `isr.Col` - `map[any]any` representing a table column
- `isr.Cols` - `[]Col` slice of table columns
- `isr.DLs` - `[]insyra.IDataList` slice of DataLists
- `isr.CSV` - CSV file or string configuration with input/output options
- `isr.JSON` - JSON file/data configuration (supports both file path and byte data)
- `isr.Excel` - Excel file configuration with input options (supports XLAM/XLSM/XLSX/XLTM/XLTX)
- `isr.Name` - Function to create named references for rows/columns

### Type Alias Details

#### CSV Structure

```go
// CSV struct supports both file path and string input
type CSV struct {
    FilePath   string      // Path to CSV file (use this OR String, not both)
    String     string      // CSV data as string (use this OR FilePath, not both)
    InputOpts  CSV_inOpts  // Options for reading CSV
    OutputOpts CSV_outOpts // Reserved for output (not used by DT.From)
}

// CSV_inOpts structure
type CSV_inOpts struct {
    FirstCol2RowNames bool   // Treat the first column as row names
    FirstRow2ColNames bool   // Treat the first row as column names
    Encoding          string // Specify input file encoding (e.g., "big5", "utf-8"), Only for FilePath input
}

// Examples:
csvFromFile := isr.CSV{FilePath: "data.csv"}
csvFromString := isr.CSV{String: "name,age\nJohn,30\nJane,25"}
```

#### JSON Structure

```go
// JSON struct supports both file path and byte data
type JSON struct {
    FilePath string // Path to JSON file (use this OR Bytes, not both)
    Bytes    []byte // JSON data as byte slice (use this OR FilePath, not both)
}
```

#### Excel Structure

```go
// Excel struct for specifying Excel file and sheet
type Excel struct {
    FilePath  string
    SheetName string
    InputOpts Excel_inOpts
}

// Excel_inOpts structure for reading Excel files
type Excel_inOpts struct {
    FirstCol2RowNames bool // Treat the first column as row names
    FirstRow2ColNames bool // Treat the first row as column names
}

// Example:
excelFile := isr.Excel{
    FilePath:  "data.xlsx",
    SheetName: "Sheet1",
    InputOpts: isr.Excel_inOpts{
        FirstCol2RowNames: true,
        FirstRow2ColNames: true,
    },
}
```

#### Row and Rows Types

```go
// Single row
row := isr.Row{
    "Name": "John",
    "Age":  30,
    "City": "NYC",
}

// Multiple rows using Rows alias
rows := isr.Rows{
    {"Name": "John", "Age": 30},
    {"Name": "Jane", "Age": 25},
    {"Name": "Bob",  "Age": 35},
}
```

#### Col and Cols Types

```go
// Single column
col := isr.Col{
    0: "John",
    1: "Jane",
    2: "Bob",
}

// Multiple columns using Cols alias
cols := isr.Cols{
    {0: "John", 1: "Jane", 2: "Bob"},     // Names column
    {0: 30, 1: 25, 2: 35},               // Ages column
    {0: "NYC", 1: "LA", 2: "Chicago"},   // Cities column
}
```

#### Name Function

```go
// Create named references for accessing rows/columns
userID := isr.Name("UserID")
email := isr.Name("Email")

// Use in DataTable operations
value := dataTable.At(userID, email)
```

#### JSON Configuration

```go
// Load from JSON file
jsonFromFile := isr.JSON{
    FilePath: "data.json",
}

// Load from JSON byte data
jsonData := []byte(`[{"name":"John","age":30},{"name":"Jane","age":25}]`)
jsonFromBytes := isr.JSON{
    Bytes: jsonData,
}

// Use in DataTable creation
dataTable1 := isr.DT.From(jsonFromFile)
dataTable2 := isr.DT.From(jsonFromBytes)
```

#### CSV Configuration

```go
// CSV from file
csvFromFile := isr.CSV{
    FilePath: "data.csv",
    InputOpts: isr.CSV_inOpts{
        FirstCol2RowNames: true,
        FirstRow2ColNames: true,
        Encoding:          "big5", // Specify file encoding
    },
    OutputOpts: isr.CSV_outOpts{
        RowNames2FirstCol: true,
        ColNames2FirstRow: true,
    },
}

// CSV from string
csvFromString := isr.CSV{
    String: "name,age,city\nJohn,30,NYC\nJane,25,LA",
    InputOpts: isr.CSV_inOpts{
        FirstCol2RowNames: false,
        FirstRow2ColNames: true,
    },
}

// Use in DataTable creation
dataTable1 := isr.DT.From(csvFromFile)
dataTable2 := isr.DT.From(csvFromString)
```

## DataList Operations

### Creating DataLists

#### Basic Creation

```go
// Create DataList with values
dataList := isr.DL.From("A", "B", "C", 1, 2, 3, 4, 5, 6)
```

**Method:**

```go
isr.DL.From(...values) *dl
```

**Description:** Use when you need this method.

**Parameters:**

- Any number of values of any type

**Returns:**

- Pointer to DataList wrapper (*dl)

**Equivalent:** `insyra.NewDataList()`

#### Creating DataLists with Of (Alias)

```go
// Create DataList with values using Of (alias for From)
dataList := isr.DL.Of("A", "B", "C", 1, 2, 3, 4, 5, 6)
```

**Method:**

```go
isr.DL.Of(...values) *dl
```

**Description:** Alias for `From()` providing a shorter method name

**Parameters:**

- Any number of values of any type

**Returns:**

- Pointer to DataList wrapper (*dl)

#### Converting Existing DataList

```go
// Convert insyra.DataList to isr.DL (recommended)
originalList := insyra.NewDataList(1, 2, 3)
isrList := isr.UseDL(originalList)

// Deprecated: Use UseDL instead
isrList := isr.PtrDL(originalList)
```

**Function:**

```go
isr.UseDL[T *insyra.DataList | dl](l T) *dl
```

**Description:** Convert between insyra and isr types (recommended)

**Parameters:**

- `*insyra.DataList` or `dl`

**Returns:**

- `*dl`

**Old Function**:

```go
isr.PtrDL[T](t T) *dl
```

**Description:** Convert between insyra and isr types (use `UseDL` instead)

**Parameters:**

- `*insyra.DataList` or `dl`

**Returns:**

- `*dl`

### DataList Methods

#### Accessing Elements

```go
// Get element at index
value := dataList.At(0)        // First element
value := dataList.At(1, 2)     // Multiple indices
```

**Method:**

```go
At(index int) any
```

**Description:** Use when you need this method.

**Parameters:**

- Integer index

**Returns:**

- Element value at specified index

**Equivalent:** `insyra.DataList.Get()`

#### Adding Elements

```go
// Append elements to DataList
dataList.Push(7, 8, 9)
```

**Method:**

```go
Push(...values) *dl
```

**Description:** Use when you need this method.

**Parameters:**

- Variable number of values

**Returns:**

- Self (for method chaining)

**Equivalent:** `insyra.DataList.Append()`

#### Method Chaining Example

```go
// All operations can be chained
result := isr.DL.From(1, 2, 3).Push(4, 5).At(3) // Returns 4
```

## DataTable Operations

### Creating DataTables

#### From 2D Slices

```go
// Create DataTable from 2D slice
dataTable := isr.DT.From([][]any{
    {"Name", "Age", "City"},
    {"John", 30, "NYC"},
    {"Jane", 25, "LA"},
})

// Supports various slice types
dataTable := isr.DT.From([][]string{
    {"A", "B"},
    {"C", "D"},
})
```

#### Empty DataTable

```go
// Create empty DataTable
dataTable := isr.DT.From(nil)
```

**Method:**

```go
isr.DT.From(nil) *dt
```

**Description:** Creates an empty DataTable that can be populated later

**Parameters:**

- `nil` value

**Returns:**

- Pointer to empty DataTable wrapper (*dt)

**Equivalent:** `insyra.NewDataTable()`

#### Creating DataTables with Of (Alias)

```go
// Create DataTable using Of (alias for From)
dataTable := isr.DT.Of(nil)

// Create from various sources
dataTable := isr.DT.Of(isr.Row{"Name": "John", "Age": 30})
dataTable := isr.DT.Of(isr.DL.From("A", "B", "C"))
```

**Method:**

```go
isr.DT.Of(item) *dt
```

**Description:** Alias for `From()` providing a shorter method name

**Parameters:**

- Same as `From()` - DataList, Row, Col, Rows, Cols, DLs, CSV, JSON, 2D slice, map, or nil

**Returns:**

- Pointer to DataTable wrapper (*dt)

#### Basic Creation from DataList

```go
// Create DataTable from single DataList
dataTable := isr.DT.From(isr.DL.From("A", "B", "C"))
```

#### From Multiple DataLists

```go
// Create DataTable from multiple DataLists as columns
dataTable := isr.DT.From(isr.DLs{
    isr.DL.From("A", "B", "C"),  // Column 1
    isr.DL.From(1, 2, 3),        // Column 2
    isr.DL.From(4, 5, 6),        // Column 3
})
```

#### From Row Data

```go
// Create DataTable from row-oriented data
dataTable := isr.DT.From(isr.Row{
    "Name": "John",
    "Age":  30,
    "City": "NYC",
})

// Multiple rows using Rows alias
dataTable := isr.DT.From(isr.Rows{
    {"Name": "John", "Age": 30},
    {"Name": "Jane", "Age": 25},
})
```

#### From Column Data

```go
// Create DataTable from column-oriented data
dataTable := isr.DT.From(isr.Col{
    isr.Name("John"): 30,
    isr.Name("Jane"): 25,
})

// Multiple columns using Cols alias
dataTable := isr.DT.From(isr.Cols{
    {0: "John", 1: "Jane"},      // Names column
    {0: 30, 1: 25},              // Ages column
})
```

#### From Files and Strings

```go
// From CSV file
dataTable := isr.DT.From(isr.CSV{
    FilePath: "data.csv",
    InputOpts: isr.CSV_inOpts{
        FirstCol2RowNames: true,
        FirstRow2ColNames: true,
    },
})

// From CSV string
dataTable := isr.DT.From(isr.CSV{
    String: "name,age,city\nJohn,30,NYC\nJane,25,LA",
    InputOpts: isr.CSV_inOpts{
        FirstRow2ColNames: true,
    },
})

// From JSON file
dataTable := isr.DT.From(isr.JSON{
    FilePath: "data.json",
})

// From JSON byte data
jsonData := []byte(`[{"name":"John","age":30},{"name":"Jane","age":25}]`)
dataTable := isr.DT.From(isr.JSON{
    Bytes: jsonData,
})

// From Excel file
dataTable := isr.DT.From(isr.Excel{
    FilePath:  "data.xlsx",
    SheetName: "Sheet1",
    InputOpts: isr.Excel_inOpts{
        FirstCol2RowNames: true,
        FirstRow2ColNames: true,
    },
})
```

#### From Maps

```go
// From string-keyed map
dataTable := isr.DT.From(map[string]any{
    "column1": "value1",
    "column2": "value2",
})

// From integer-keyed map
dataTable := isr.DT.From(map[int]any{
    0: "value1",
    1: "value2",
})
```

#### Converting Existing DataTable

```go
// Convert insyra.DataTable to isr.DT (recommended)
originalTable := insyra.NewDataTable()
isrTable := isr.UseDT(originalTable)

// Deprecated: Use UseDT instead
isrTable := isr.PtrDT(originalTable)
```

**Function:**

```go
isr.UseDT[T *insyra.DataTable | dt](t T) *dt
```

**Description:** Convert between insyra and isr types (recommended)

**Parameters:**

- `*insyra.DataTable` or `dt`

**Returns:**

- `*dt`

**Old Function**:

```go
isr.PtrDT[T](t T) *dt
```

**Description:** Convert between insyra and isr types (use `UseDT` instead)

**Parameters:**

- `*insyra.DataTable` or `dt`

**Returns:**

- `*dt`

### DataTable Methods

#### Accessing Rows

```go
// Get row by index
row := dataTable.Row(0)              // First row (returns *dl)
row := dataTable.Row(isr.Name("ID")) // Row by name
```

**Method:**

```go
Row(row any) *dl
```

**Description:** Use when you need this method.

**Parameters:**

- `int` (row number) or `isr.Name` (row name)

**Returns:**

- DataList containing row data

**Equivalent:** `insyra.DataTable.GetRow()`

#### Accessing Columns

```go
// Get column by index/name
col := dataTable.Col(0)              // First column (returns *dl)
col := dataTable.Col("Name")         // Column by string name
col := dataTable.Col(isr.Name("ID")) // Column by Name type
```

**Method:**

```go
Col(col any) *dl
```

**Description:** Use when you need this method.

**Parameters:**

- `int`, `string`, or `isr.Name`

**Returns:**

- DataList containing column data

**Equivalent:** `insyra.DataTable.GetCol()`

#### Accessing Individual Elements

```go
// Get element at row, column
value := dataTable.At(0, 1)                    // Row 0, Column 1
value := dataTable.At(0, "Name")               // Row 0, Column "Name"
value := dataTable.At(isr.Name("ID"), "Name")  // Named row and column
```

**Method:**

```go
At(row any, col any) any
```

**Description:** Use when you need this method.

**Parameters:**

- Row index (`int` or `isr.Name`) and column index (`int`, `string`, or `isr.Name`)

**Returns:**

- Element value

**Equivalent:** `insyra.DataTable.GetElement()`

#### Adding Data

```go
// Add new rows
dataTable.Push(isr.Row{    "Name": "Bob",
    "Age": 35,
})

// Add multiple rows using Rows alias
dataTable.Push(isr.Rows{
    {"Name": "Alice", "Age": 28},
    {"Name": "Charlie", "Age": 42},
})

// Add columns
dataTable.Push(isr.Col{
    0: "Engineer",
    1: "Designer",
})

// Add DataLists as columns
dataTable.Push(isr.DL.From("Sales", "Marketing", "Engineering"))

// Add multiple DataLists
dataTable.Push([]*insyra.DataList{
    insyra.NewDataList("A", "B"),
    insyra.NewDataList(1, 2),
})

// Add DLs (slice of DataLists)
dataTable.Push(isr.DLs{
    isr.DL.From("X", "Y"),
    isr.DL.From(10, 20),
})
```

**Method:**

```go
Push(data any) *dt
```

**Description:** Append data to existing DataTable

**Parameters:**

- `Row`, `[]Row`, `Col`, `[]Col`, `*dl`, `*insyra.DataList`, `[]*insyra.DataList`, `[]dl`, or `DLs`

**Returns:**

- Self (for method chaining)

#### Executing CCL Statements

```go
// Execute CCL statements on DataTable
dataTable.CCL("A = A * 2")

// Create new column using CCL
dataTable.CCL("NEW('total') = A + B + C")

// Row access and all-column reference
dataTable.CCL("NEW('first_row') = @.0")
dataTable.CCL("NEW('base_val') = A.0")

// Multiple CCL statements
dataTable.CCL(`
    A = A * 10
    B = B + 5
    NEW('sum') = A + B
`)

// Or use semicolons for multiple statements
dataTable.CCL("A = A + 1; NEW('doubled') = A * 2")

// Method chaining with CCL
result := isr.DT.From(isr.Rows{
    {"A": 1, "B": 2},
    {"A": 3, "B": 4},
}).CCL("NEW('newCol') = [A] + [B]").Col(isr.Name("newCol"))
```

**Method:**

```go
CCL(cclStatements string) *dt
```

**Description:** Execute CCL statements to modify or create columns

**Parameters:**

- CCL statement string (supports assignment syntax and NEW function)

**Returns:**

- Self (for method chaining)

**Equivalent:** `insyra.DataTable.ExecuteCCL()`

> **Note**: For detailed CCL syntax and features, see the [CCL Documentation](CCL.md).

## Advanced Features

### Named Elements

```go
// Use isr.Name for named access
namedRow := dataTable.Row(isr.Name("UserID"))
namedCol := dataTable.Col(isr.Name("Email"))
namedValue := dataTable.At(isr.Name("User1"), isr.Name("Email"))
```

### Method Chaining Examples

```go
// Complex chaining operations
result := isr.DT.From(isr.DLs{
    isr.DL.From("A", "B", "C"),
    isr.DL.From(1, 2, 3),
}).Push(isr.Row{"A": "D", 1: 4}).At(3, 0) // Returns "D"

// DataList chaining
processed := isr.DL.From(1, 2, 3).Push(4, 5).At(4) // Returns 5
```

## Type Reference

### Core Types

- `dl` - DataList wrapper struct containing `*insyra.DataList`
- `dt` - DataTable wrapper struct containing `*insyra.DataTable`

### Input Types

- `Row` - `map[any]any` for row data
- `Rows` - `[]Row` for multiple rows
- `Col` - `map[any]any` for column data
- `Cols` - `[]Col` for multiple columns
- `DLs` - `[]insyra.IDataList` for multiple DataLists
- `CSV` - Struct for CSV file or string loading with input/output options
- `JSON` - Struct for JSON file/data loading (supports both file path and byte data)
- `Excel` - Struct for Excel file loading with input options

### Key Functions

- `PtrDL[T]` - Convert to `*dl` (deprecated, use `UseDL` instead)
- `PtrDT[T]` - Convert to `*dt` (deprecated, use `UseDT` instead)
- `UseDL[T]` - Convert to `*dl` (recommended)
- `UseDT[T]` - Convert to `*dt` (recommended)
- `Name(string)` - Create named reference

## Error Handling

All methods use `insyra.LogFatal()` for error handling. Invalid operations will terminate the program with descriptive error messages.

## Best Practices

1. **Use method chaining** for fluent operations
2. **Use `isr.DL.From()`** instead of `isr.DL{}.From()`
3. **Use `isr.DT.From()`** instead of `isr.DT{}.From()`
4. **Use `UseDL/UseDT`** when converting from insyra types (recommended over `PtrDL/PtrDT`)
5. **Use `isr.Name()`** for named element access

## Quick Reference

### DataList Quick Reference

```go
// Create and manipulate DataList
dl := isr.DL.From(1, 2, 3)           // Create
dl.Push(4, 5)                        // Add elements
value := dl.At(0)                    // Get element
chained := isr.DL.From(1).Push(2).At(1) // Method chaining
```

### DataTable Quick Reference

```go
// Create DataTable
dt := isr.DT.From(isr.DL.From("A", "B", "C"))

// Access data
row := dt.Row(0)                     // Get row
col := dt.Col("Name")                // Get column
value := dt.At(0, "Name")            // Get specific element

// Add data
dt.Push(isr.Row{"Name": "John"})     // Add row
dt.Push(isr.Col{0: "Value"})         // Add column
```

### File and String Operations

```go
// Load from files
csvTable := isr.DT.From(isr.CSV{FilePath: "data.csv"})
jsonTable := isr.DT.From(isr.JSON{FilePath: "data.json"})

// Load from strings/byte data
csvFromString := isr.DT.From(isr.CSV{String: "name,age\nJohn,30\nJane,25"})
jsonData := []byte(`[{"name":"John","age":30}]`)
jsonFromBytes := isr.DT.From(isr.JSON{Bytes: jsonData})
```

### Conversion Operations

```go
// Convert insyra types to isr types (recommended)
isrDL := isr.UseDL(insyraDataList)
isrDT := isr.UseDT(insyraDataTable)

// Deprecated: Use UseDL/UseDT instead
isrDL := isr.PtrDL(insyraDataList)
isrDT := isr.PtrDT(insyraDataTable)
```
