# isr Package Documentation

## Overview

The `isr` package provides a simplified, method-chaining syntax for the `insyra` data analysis library.

**Package Name**: `isr` (stands for "insyra" - first letters of three syllables)

**Purpose**: Syntax sugar and fluent API for insyra data structures

**Key Feature**: Full method chaining support

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
- `isr.Col` - `map[any]any` representing a table column
- `isr.DLs` - `[]*DL` slice of DataLists
- `isr.CSV` - CSV file configuration
- `isr.JSON` - JSON file configuration

## DataList Operations

### Creating DataLists

#### Basic Creation

```go
// Create DataList with values
dataList := isr.DL.From("A", "B", "C", 1, 2, 3, 4, 5, 6)
```

**Method**: `isr.DL.From(...values) *dl`

- **Input**: Any number of values of any type
- **Output**: Pointer to DataList wrapper (*dl)
- **Equivalent**: `insyra.NewDataList()`

#### Converting Existing DataList

```go
// Convert insyra.DataList to isr.DL
originalList := insyra.NewDataList(1, 2, 3)
isrList := isr.PtrDL(originalList)
```

**Function**: `isr.PtrDL[T](t T) *dl`

- **Input**: `*insyra.DataList` or `dl`
- **Output**: `*dl`
- **Purpose**: Convert between insyra and isr types

### DataList Methods

#### Accessing Elements

```go
// Get element at index
value := dataList.At(0)        // First element
value := dataList.At(1, 2)     // Multiple indices
```

**Method**: `At(...indices) any`

- **Input**: Variable number of integer indices
- **Output**: Element value at specified index
- **Equivalent**: `insyra.DataList.Get()`

#### Adding Elements

```go
// Append elements to DataList
dataList.Push(7, 8, 9)
```

**Method**: `Push(...values) *dl`

- **Input**: Variable number of values
- **Output**: Self (for method chaining)
- **Equivalent**: `insyra.DataList.Append()`

#### Method Chaining Example

```go
// All operations can be chained
result := isr.DL.From(1, 2, 3).Push(4, 5).At(3) // Returns 4
```

## DataTable Operations

### Creating DataTables

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

// Multiple rows
dataTable := isr.DT.From([]isr.Row{
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

// Multiple columns
dataTable := isr.DT.From([]isr.Col{
    {0: "John", 1: "Jane"},      // Names column
    {0: 30, 1: 25},              // Ages column
})
```

#### From Files

```go
// From CSV file
dataTable := isr.DT.From(isr.CSV{
    FilePath: "data.csv",
    InputOpts: isr.CSV_inOpts{
        FirstCol2RowNames: true,
        FirstRow2ColNames: true,
    },
})

// From JSON file
dataTable := isr.DT.From(isr.JSON{
    FilePath: "data.json",
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
// Convert insyra.DataTable to isr.DT
originalTable := insyra.NewDataTable()
isrTable := isr.PtrDT(originalTable)
```

**Function**: `isr.PtrDT[T](t T) *dt`

- **Input**: `*insyra.DataTable` or `dt`
- **Output**: `*dt`
- **Purpose**: Convert between insyra and isr types

### DataTable Methods

#### Accessing Rows

```go
// Get row by index
row := dataTable.Row(0)              // First row (returns *dl)
row := dataTable.Row(isr.Name("ID")) // Row by name
```

**Method**: `Row(index) *dl`

- **Input**: `int` (row number) or `isr.Name` (row name)
- **Output**: DataList containing row data
- **Equivalent**: `insyra.DataTable.GetRow()`

#### Accessing Columns

```go
// Get column by index/name
col := dataTable.Col(0)              // First column (returns *dl)
col := dataTable.Col("Name")         // Column by string name
col := dataTable.Col(isr.Name("ID")) // Column by Name type
```

**Method**: `Col(index) *dl`

- **Input**: `int`, `string`, or `isr.Name`
- **Output**: DataList containing column data
- **Equivalent**: `insyra.DataTable.GetCol()`

#### Accessing Individual Elements

```go
// Get element at row, column
value := dataTable.At(0, 1)                    // Row 0, Column 1
value := dataTable.At(0, "Name")               // Row 0, Column "Name"
value := dataTable.At(isr.Name("ID"), "Name")  // Named row and column
```

**Method**: `At(row, col) any`

- **Input**: Row index (`int` or `isr.Name`) and column index (`int`, `string`, or `isr.Name`)
- **Output**: Element value
- **Equivalent**: `insyra.DataTable.GetElement()`

#### Adding Data

```go
// Add new rows
dataTable.Push(isr.Row{
    "Name": "Bob",
    "Age": 35,
})

// Add multiple rows
dataTable.Push([]isr.Row{
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
```

**Method**: `Push(data) *dt`

- **Input**: `Row`, `[]Row`, `Col`, `[]Col`, `*dl`, `*insyra.DataList`, or slices thereof
- **Output**: Self (for method chaining)
- **Purpose**: Append data to existing DataTable

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
- `Col` - `map[any]any` for column data
- `DLs` - `[]*dl` for multiple DataLists
- `CSV` - Struct for CSV file loading
- `JSON` - Struct for JSON file loading

### Key Functions

- `PtrDL[T]` - Convert to `*dl`
- `PtrDT[T]` - Convert to `*dt`
- `Name(string)` - Create named reference

## Error Handling

All methods use `insyra.LogFatal()` for error handling. Invalid operations will terminate the program with descriptive error messages.

## Best Practices

1. **Use method chaining** for fluent operations
2. **Use `isr.DL.From()`** instead of `isr.DL{}.From()`
3. **Use `isr.DT.From()`** instead of `isr.DT{}.From()`
4. **Use `PtrDL/PtrDT`** when converting from insyra types
5. **Use `isr.Name()`** for named element access

## Quick Reference

### DataList Operations

```go
// Create and manipulate DataList
dl := isr.DL.From(1, 2, 3)           // Create
dl.Push(4, 5)                        // Add elements
value := dl.At(0)                    // Get element
chained := isr.DL.From(1).Push(2).At(1) // Method chaining
```

### DataTable Operations

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

### File Operations

```go
// Load from files
csvTable := isr.DT.From(isr.CSV{FilePath: "data.csv"})
jsonTable := isr.DT.From(isr.JSON{FilePath: "data.json"})
```

### Conversion Operations

```go
// Convert insyra types to isr types
isrDL := isr.PtrDL(insyraDataList)
isrDT := isr.PtrDT(insyraDataTable)
```
