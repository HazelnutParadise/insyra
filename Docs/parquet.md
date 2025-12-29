# [ parquet ] Package

The `parquet` package provides read and write support for the Apache Parquet file format, deeply integrated with Insyra's `DataTable` and `DataList`.

## Table of Contents

- [Data Structures](#data-structures)
  - [ReadOptions](#readoptions)
  - [ReadColumnOptions](#readcolumnoptions)
  - [FileInfo](#fileinfo)
- [Main Functions](#main-functions)
  - [Inspect](#inspect)
  - [Read](#read)
  - [Write](#write)
  - [Stream](#stream)
  - [ReadColumn](#readcolumn)
- [CCL Support](#ccl-support)
  - [FilterWithCCL](#filterwithccl)
  - [ApplyCCL](#applyccl)
  - [Type Constraints](#type-constraints)
- [Examples](#examples)

## Data Structures

### ReadOptions

Options for configuring Parquet file reading.

```go
type ReadOptions struct {
    Columns   []string // Column names to read; if empty, all columns are read
    RowGroups []int    // RowGroup indices to read; if empty, all RowGroups are read
}
```

### ReadColumnOptions

Options specifically for the `ReadColumn` function.

```go
type ReadColumnOptions struct {
    RowGroups []int // RowGroup indices to read; if empty, all RowGroups are read
    MaxValues int64 // Maximum number of values to read; 0 means no limit. If exceeded, an error is returned to prevent memory overflow.
}
```

### FileInfo

Contains metadata information of a Parquet file.

```go
type FileInfo struct {
    NumRows      int64             // Total number of rows
    NumRowGroups int               // Number of RowGroups
    Version      string            // Parquet version
    CreatedBy    string            // Writer information
    Metadata     map[string]string // Key-value metadata
    Columns      []ColumnInfo      // Column information
    RowGroups    []RowGroupInfo    // RowGroup information
}
```

## Main Functions

### Inspect

Inspects the metadata of a Parquet file.

```go
func Inspect(path string) (FileInfo, error)
```

### Read

Reads a Parquet file into an `insyra.DataTable` all at once.

```go
func Read(ctx context.Context, path string, opt ReadOptions) (*insyra.DataTable, error)
```

### Write

Writes an `insyra.IDataTable` to a Parquet file.

```go
func Write(dt insyra.IDataTable, path string) error
```

### Stream

Streams a Parquet file, returning a channel that receives `*insyra.DataTable` batches.

```go
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error)
```

### ReadColumn

Reads data from a single column in a Parquet file, returning an `insyra.DataList`.

```go
func ReadColumn(ctx context.Context, path string, column string, opt ReadColumnOptions) (*insyra.DataList, error)
```

## CCL Support

The `parquet` package provides CCL (Column Calculation Language) support for direct manipulation of Parquet files without loading the entire dataset into memory.

> [!NOTE]
> **⚠️ Important Note on Type Constraints:**
>
> Due to the nature of Parquet format, **each column must have a consistent data type**. This means CCL operations in Parquet may behave differently from DataTable operations in the following ways:
>
> - When creating new columns or modifying existing ones, ensure that the resulting values maintain type consistency within each column
> - Type coercion may occur automatically to maintain column type consistency
> - Operations that would create mixed types in a column may result in errors or unexpected behavior
> - This is a fundamental constraint of the Parquet format, not a limitation of the CCL implementation

### FilterWithCCL

Applies a CCL filter expression to a Parquet file and returns filtered results as a `DataTable`. The filter expression should evaluate to a boolean value for each row.

```go
func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error)
```

**Parameters:**

- `ctx`: Context for cancellation
- `path`: Path to the input Parquet file
- `filterExpr`: CCL expression that evaluates to boolean (e.g., `"(A > 100) && (B == 'active')"`)

**Returns:**

- A new `DataTable` containing only rows that satisfy the filter condition
- The original Parquet file is **not modified**

**Example:**

```go
// Filter rows where column A > 100 and column B equals 'active'
filtered, err := parquet.FilterWithCCL(ctx, "data.parquet", "(A > 100) && (B == 'active')")
if err != nil {
    panic(err)
}
filtered.Show()
```

### ApplyCCL

Applies CCL expressions directly to a Parquet file in streaming mode, processing data batch by batch to minimize memory usage. The CCL script can contain multiple statements separated by semicolons.

```go
func ApplyCCL(ctx context.Context, path string, cclScript string) error
```

**Parameters:**

- `ctx`: Context for cancellation
- `path`: Path to the Parquet file (will be modified in-place)
- `cclScript`: CCL script containing one or more statements separated by `;` or newlines

**Important:**

- The input file **will be overwritten** with the transformed data
- Processing is done in batches to handle large files efficiently
- Supports creating new columns with `NEW()`, but modifying existing columns may not work.

**Example:**

```go
// Create a new column Sum as sum of Price1 and Price2
err := parquet.ApplyCCL(ctx, "data.parquet", `
    NEW('Sum') = ['Price1'] + ['Price2']
`)
if err != nil {
    panic(err)
}
```

### Type Constraints

When using CCL with Parquet files, be aware of these type-related considerations:

1. **Column Type Consistency**: Each column must maintain a single data type throughout. Mixed-type columns are not supported.

2. **Type Inference**: When creating new columns with `NEW()`, the type is determined from the first batch of data processed.

3. **Type Coercion**: Operations may automatically coerce types to maintain consistency. For example:
   - Numeric operations on integer columns may produce float results
   - String concatenation with numbers will convert numbers to strings

4. **Differences from DataTable CCL**:
   - DataTable allows more flexible type handling per cell
   - Parquet enforces strict column-level typing
   - Some CCL operations that work on DataTable may need adjustment for Parquet

**Best Practices:**

- Test CCL expressions on a small sample file first
- Explicitly handle type conversions in your CCL expressions when needed
- Be aware that aggregate functions must return consistent types

## Examples

### Reading a Parquet File

```go
package main

import (
    "context"
    "fmt"
    "github.com/HazelnutParadise/insyra/parquet"
)

func main() {
    ctx := context.Background()
    dt, err := parquet.Read(ctx, "data.parquet", parquet.ReadOptions{})
    if err != nil {
        panic(err)
    }
    dt.Show()
}
```

### Writing a Parquet File

```go
package main

import (
 "github.com/HazelnutParadise/insyra/isr"
 "github.com/HazelnutParadise/insyra/parquet"
)

func main() {
    dt := isr.DT.Of(isr.DLs{
        isr.DL.Of(1, 2, 3).SetName("ID"),
        isr.DL.Of("A", "B", "C").SetName("Name"),
    })

    err := parquet.Write(dt, "output.parquet")
    if err != nil {
     panic(err)
    }
}
```

### Streaming Read

```go
package main

import (
    "context"
    "fmt"
    "github.com/HazelnutParadise/insyra/parquet"
)

func main() {
    ctx := context.Background()
    dtChan, errChan := parquet.Stream(ctx, "large_data.parquet", parquet.ReadOptions{}, 1000)

    for {
        select {
        case dt, ok := <-dtChan:
            if !ok {
                return
            }
            numRows, _ := dt.Size()
            fmt.Printf("Batch read, rows: %d\n", numRows)
            dt.Show()
        case err := <-errChan:
            if err != nil {
                panic(err)
            }
            return
        }
    }
}
```

### Using CCL to Filter Data

```go
package main

import (
    "context"
    "github.com/HazelnutParadise/insyra/parquet"
)

func main() {
    ctx := context.Background()
    
    // Filter products with price > 100 and in_stock == true
    filtered, err := parquet.FilterWithCCL(
        ctx,
        "products.parquet",
        "(['price'] > 100) && (['in_stock'] == true)",
    )
    if err != nil {
        panic(err)
    }
    
    filtered.Show()
}
```

### Using CCL to Transform Data

```go
package main

import (
    "context"
    "github.com/HazelnutParadise/insyra/parquet"
)

func main() {
    ctx := context.Background()
    
    // Apply multiple CCL transformations:
    // 1. Create a new column 'total' as price * quantity
    // 2. Apply 10% discount to all prices
    // 3. Update status based on stock level
    err := parquet.ApplyCCL(ctx, "orders.parquet", `
        NEW('total') = ['price'] * ['quantity']
        NEW('new_price') = ['price'] * 0.9
        NEW('status') = IF(['stock'] > 0, 'available', 'out_of_stock')
    `)
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Transformations applied successfully!")
}
```
