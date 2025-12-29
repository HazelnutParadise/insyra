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
    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/parquet"
)

func main() {
    dt := isr.DT.Of(
        isr.DL.Of(1, 2, 3).SetName("ID"),
        isr.DL.Of("A", "B", "C").SetName("Name"),
    )

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
