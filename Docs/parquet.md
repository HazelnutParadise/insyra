# [ parquet ] Package

The `parquet` package provides read and write support for the Apache Parquet file format, deeply integrated with Insyra's `DataTable` and `DataList`.

## Table of Contents

- [Data Structures](#data-structures)
  - [ReadOptions](#readoptions)
  - [ReadColumnOptions](#readcolumnoptions)
  - [WriteOptions and Compression](#writeoptions-and-compression)
  - [FileInfo](#fileinfo)
- [Main Functions](#main-functions)
  - [Inspect](#inspect)
  - [Read](#read)
  - [Write](#write)
  - [WriteContext, WriteToContext](#writecontext-writetocontext)
  - [Stream](#stream)
  - [ReadFrom, StreamFrom, WriteTo](#readfrom-streamfrom-writeto--from-and-to-any-source)
  - [ReadColumn](#readcolumn)
- [CCL Support](#ccl-support)
  - [Batches of 1,000 rows](#batches-of-1000-rows)
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
    MaxValues int64 // Maximum number of values to read; 0 means no limit. The row count of the selected RowGroups is checked from the file metadata before any value is read; if it exceeds MaxValues an error naming both numbers is returned.
}
```

### WriteOptions and Compression

Settings for `Write`, `WriteTo`, `WriteContext` and `WriteToContext`. The zero value writes what `Write` always wrote: uncompressed, with up to 1,048,576 rows in a row group.

```go
type WriteOptions struct {
    Compression  Compression // Codec for every column, CompressionNone when unset
    RowGroupSize int         // Most rows in one row group, 1,048,576 when 0
}

type Compression int

const (
    CompressionNone   Compression = iota // uncompressed (the zero value)
    CompressionSnappy
    CompressionGzip
    CompressionBrotli
    CompressionZstd
)
```

`RowGroupSize` sets how the file is split into row groups, the units a reader can skip with `ReadOptions.RowGroups`. The Arrow writer insyra uses caps a row group at 67,108,864 rows, so a larger value writes groups of that size. A negative `RowGroupSize`, a `Compression` that is not one of the constants above, or more than one `WriteOptions` is an error, returned before anything is written.

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

```go
func Inspect(path string) (FileInfo, error)
```

**Description:** Inspects the metadata of a Parquet file.

**Parameters:**

- `path`: File path to use. Type: `string`.

**Returns:**

- `FileInfo`: Return value.
- `error`: Error when the operation fails.

### Read

```go
func Read(ctx context.Context, path string, opt ReadOptions) (*insyra.DataTable, error)
```

**Description:** Reads a Parquet file into an `insyra.DataTable` all at once. A file the Arrow reader cannot make sense of, such as one whose data pages and footer came from different writes, is an error saying it is not a readable Parquet file, never a panic. The same holds for `ReadFrom`, `ReadColumn`, `Inspect`, `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL`, and `ApplyCCL` then leaves the file as it was. A file with a damaged row group, such as a page header overwritten by a failed copy, is an error too: the Arrow reader takes such a page for the end of its row group and goes on with the next, so every reader compares the rows it read with the row count in the file's metadata and fails when they differ, naming both counts and the row groups that did not read in full. Reading only the intact row groups through `ReadOptions.RowGroups` still works.

**Parameters:**

- `ctx`: Context for cancellation or timeouts. Type: `context.Context`.
- `path`: File path to use. Type: `string`.
- `opt`: Input value for `opt`. Type: `ReadOptions`.

**Returns:**

- `*insyra.DataTable`: Return value.
- `error`: Error when the operation fails.

**Column types:** the table below is what each Arrow column type in the file
becomes in Go. A type that is not listed has no faithful Go representation, so
its cells are read as `nil` and the reason, naming the column and the Arrow
type, is recorded on the returned table's `Err()`. The rest of the file still
reads normally, and `ReadOptions.Columns` can be used to skip such a column
entirely.

| Arrow column type | Go value |
| --- | --- |
| `Int8`, `Int16`, `Int32`, `Int64` | `int8`, `int16`, `int32`, `int64` |
| `Uint8`, `Uint16`, `Uint32`, `Uint64` | `uint8`, `uint16`, `uint32`, `uint64` |
| `Float32`, `Float64` | `float32`, `float64` |
| `Bool` | `bool` |
| `String`, `LargeString` | `string` |
| `Binary`, `LargeBinary`, `FixedSizeBinary` | `[]byte`, so a binary column is never mistaken for a text one; `Show` prints each cell as hex, shortened for a value longer than 20 bytes (for example `30313233343536373839... (26 bytes)`) |
| `Timestamp` | `time.Time` |
| `Date32`, `Date64` | `time.Time` at UTC midnight |
| `Decimal128`, `Decimal256` | `decimal.Decimal` ([go-decimal](https://github.com/TimLai666/go-decimal)), exact |
| `Null` | `nil` in every cell, which is all such a column holds; no reason is recorded |
| anything else | `nil`, with the reason on `Err()` |

A `Decimal` keeps the file's own unscaled integer and scale, so nothing is
rounded, and it sorts by value rather than by the text of its digits. It is a
number to `Mean`, `Sum` and the rest of the numeric path, which read it
through its own text; the cell keeps its exact value, and the rounding to
about 16 significant digits happens only in the float result. See [Exact Decimals](Decimal.md).

A binary column exports to JSON as base64, which is what `encoding/json` does
with a `[]byte`, so nothing is lost. Writing the table back to Parquet keeps
it a binary column.

A dictionary-encoded column reads as the values it holds, in the Go type the
table above gives their Arrow type, so a pandas `category` column of strings
reads as strings. That holds whether the reader materialises the column or, for
a file that stores its Arrow schema, hands it over as an Arrow dictionary. A
dictionary whose values have no Go representation reads as `nil`, with the
reason on `Err()`.

### Write

```go
func Write(dt insyra.IDataTable, path string, opts ...WriteOptions) error
```

**Description:** Writes an `insyra.IDataTable` to a Parquet file. The data goes to a temporary file with a name of its own in the same directory, which is renamed into place, so a failure part-way never leaves a truncated file at `path`, and two writes to the same path at once never mix their bytes. `Write` is `WriteContext` with `context.Background()`.

**Parameters:**

- `dt`: Input value for `dt`. Type: `insyra.IDataTable`.
- `path`: File path to use. Type: `string`.
- `opts`: Optional. Compression and row group size, described under [WriteOptions](#writeoptions-and-compression). At most one.

**Returns:**

- `error`: Error when the operation fails.

```go
err := parquet.Write(dt, "sales.parquet", parquet.WriteOptions{
    Compression:  parquet.CompressionZstd,
    RowGroupSize: 100_000,
})
```

### WriteContext, WriteToContext

```go
func WriteContext(ctx context.Context, dt insyra.IDataTable, path string, opts ...WriteOptions) error
func WriteToContext(ctx context.Context, dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error
```

**Description:** `Write` and `WriteTo` with a context, so a large write can be cancelled or given a deadline. The context is checked before the table is converted, between columns while converting, and before each row group, and `WriteContext` checks it once more before the finished file replaces `path`. A cancelled call returns the context's error. `WriteContext` then leaves the file at `path` as it was. `WriteToContext` stops without writing the Parquet footer, so what already reached `w` is not a readable file. A row group that has started is finished before the next check, so a smaller `RowGroupSize` makes a write stop sooner. A nil `ctx` is an error.

### Stream

```go
func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error]
```

**Description:** Streams a Parquet file batch by batch. Range over the result with `for dt, err := range`: each batch arrives as a `*insyra.DataTable` with a nil error, and a failure arrives once, as a nil table with the error, and ends the loop. Breaking out of the loop early stops the reader, so nothing is left running and there is nothing to cancel. Cancelling `ctx` ends the loop with the context's error.

**Parameters:**

- `ctx`: Context for cancellation or timeouts. Type: `context.Context`.
- `path`: File path to use. Type: `string`.
- `opt`: Input value for `opt`. Type: `ReadOptions`.
- `batchSize`: Input value for `batchSize`. Type: `int`.

**Returns:**

- `iter.Seq2[*insyra.DataTable, error]`: The batches, ready to range over.


### ReadFrom, StreamFrom, WriteTo — from and to any source

```go
func ReadFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions) (*insyra.DataTable, error)
func StreamFrom(ctx context.Context, r io.ReaderAt, size int64, opt ReadOptions, batchSize int) iter.Seq2[*insyra.DataTable, error]
func WriteTo(dt insyra.IDataTable, w io.Writer, opts ...WriteOptions) error
```

**Description:** Read and write Parquet that is not a file on disk — an S3 object, an upload, bytes in memory. `Read`, `Stream` and `Write` call these, so a path and a source over the same bytes give the same result. Reading takes an `io.ReaderAt` and the total size rather than a plain `io.Reader`, because Parquet keeps its index at the end of the file and the reader has to seek to it; `*os.File`, `*bytes.Reader` and S3 range readers all qualify. `WriteTo` takes the same `WriteOptions` as `Write`, and does not close `w`: the caller owns it.

**Example:**

```go
data, _ := os.ReadFile("sales.parquet") // or bytes from anywhere
dt, err := parquet.ReadFrom(ctx, bytes.NewReader(data), int64(len(data)), parquet.ReadOptions{})

var buf bytes.Buffer
err = parquet.WriteTo(dt, &buf)
```

### ReadColumn

```go
func ReadColumn(ctx context.Context, path string, column string, opt ReadColumnOptions) (*insyra.DataList, error)
```

**Description:** Reads data from a single column in a Parquet file, returning an `insyra.DataList`. The column is selected by its Parquet leaf name, as `ReadOptions.Columns` selects them, so a nested column (`List`, `Struct`, `Map`), whose leaf is not named after the field, cannot be selected by name: the call reports that the column is not found. When `opt.MaxValues > 0`, the row count of the selected row groups (all when none are selected) is taken from the metadata first and the call is refused before reading if it exceeds the limit.

**Parameters:**

- `ctx`: Context for cancellation or timeouts. Type: `context.Context`.
- `path`: File path to use. Type: `string`.
- `column`: Input value for `column`. Type: `string`.
- `opt`: Input value for `opt`. Type: `ReadColumnOptions`.

**Returns:**

- `*insyra.DataList`: Return value.
- `error`: Error when the operation fails.

## CCL Support

The `parquet` package provides CCL (Column Calculation Language) support for direct manipulation of Parquet files without loading the entire dataset into memory.

> [!IMPORTANT]
> **⚠️ Important Note on Type Constraints:**
>
> A Parquet column holds one type, while a `DataTable` column can hold values of any kind. `ApplyCCL` settles the type of every column a statement writes so that no value is lost:
>
> - a column the file had keeps its type when every value written into it fits, such as whole numbers in an integer column; a fraction widens an integer column to `float64` rather than being cut off;
> - a column the script creates, or a column whose new values do not fit its type, takes the type `Write` gives a table's column: CCL computes numbers as `float64`, and a column mixing text, numbers or booleans is text, so its values read back as strings;
> - see [Type Constraints](#type-constraints) for the rest.

### Batches of 1,000 rows

`FilterWithCCL` and `ApplyCCL` read the file 1,000 rows at a time and give the answer the same expression gives on the file loaded with `Read`:

- an aggregate such as `AVG(A)`, `SUM(A:C)` or `STDEV(A - 1)` is computed over the whole file;
- the row index `#` is the row's position in the file, counted from 0;
- a reference to a fixed row or range of rows, such as `A.0`, `@.1500` or `A.0:1999`, reads those rows of the file.

For example, on a file whose column `A` holds 1 to 2,500, `FilterWithCCL(ctx, path, "A > AVG(A)")` keeps the 1,250 rows from 1,251 on, and `"# == 0"` keeps the first row only. In `ApplyCCL`, each statement sees the file as the statements before it leave it, so `NEW('c') = A - AVG(A); NEW('d') = ['c'] / SUM(['c'])` divides by the sum of the new column.

The aggregates computed this way are `SUM`, `AVG`, `COUNT`, `MIN`, `MAX`, `VAR`, `VARP`, `STDEV` and `STDEVP`. Before any row is evaluated, the file is read again for them, keeping only running totals, so memory still holds one batch at a time:

- each extra read feeds one argument of an aggregate, and each column of a range or of `@` is an argument of its own;
- `VAR`, `VARP`, `STDEV` and `STDEVP` read each argument twice;
- aggregates side by side share their reads, and an aggregate inside another waits for the inner one;
- fixed rows take a read of their own, separate from the aggregates', and one more when they sit inside an aggregate or another fixed row; the rows they name are kept in memory, so `A.0:1999` holds 2,000 rows.

`A > AVG(A)` reads the file twice in all, `A > STDEV(A)` and `A > AVG(A - AVG(A))` three times. An expression that needs none of this is read once, as before. In `ApplyCCL`, every extra read for a statement applies the statements before it to each batch again.

A sequence function, `LAG`, `LEAD`, `DIFF`, `PCT_CHANGE`, `CUMSUM`, `CUMPROD`, `CUMMAX`, `CUMMIN`, `ROLLING_SUM`, `ROLLING_MEAN`, `ROLLING_MIN`, `ROLLING_MAX` or `ROLLING_STD`, works when it is the whole filter expression or the whole right-hand side of a `NEW` or an assignment: `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")` writes the running sum of the whole file. Its column argument can be any expression of the row, such as `A - AVG(A)`, and its other arguments must be constants. It is computed as the file is read, with no extra read of its own, and holds only what crosses a batch boundary:

- the last `n` values of its column for `LAG(x, n)`, `DIFF(x, n)` and `PCT_CHANGE(x, n)`, and the last `w - 1` for `ROLLING_*(x, w)`;
- the running value for `CUMSUM`, `CUMPROD`, `CUMMAX` and `CUMMIN`;
- for `LEAD(x, n)`, the next `n` rows of the file, every column of them: a row waits until the value it needs has been read.

A part that cannot be computed, such as `STDEV(A)` over a single value or `A.5000` in a file of 2,500 rows, fails only the rows that reach it, as it does on a loaded table: `IF(COUNT(A) >= 2, A > STDEV(A), TRUE)` keeps every row of a one-row file.

Anything else that reads beyond the current row is computed by reading the whole of the columns the expression reads, and only those, into memory, and evaluating the expression the way `AddColUsingCCL` and `ExecuteCCL` do on a loaded table, so its answer is the table's, and it fails where the table fails, for the same reason:

- `MEDIAN`, or an aggregate registered with `RegisterAggregateFunction` from `engine/ccl`, including one registered under a built-in name;
- a row reference computed from the current row, such as `A.(# - 1)`;
- a column range inside an expression an aggregate is computed over, such as `COUNT(IF(A > 0, A:B, 0))`;
- a sequence function anywhere but as the whole expression: inside another sequence function or an aggregate, such as `LAG(CUMSUM(A), 1)` or `SUM(CUMSUM(A))`, or inside another expression, such as `CUMSUM(A) + 1`, which fails here as it fails on a loaded table.

`A > MEDIAN(A)` on a file with ten columns holds column `A`, whole, and nothing else. `@` reads every column, so such an expression that uses `@` holds the whole file. It costs one more read of the file. In `ApplyCCL`, such a statement is computed before the file is written, against the file as the statements before it leave it, and the column it writes is held until the file is written.

### FilterWithCCL

```go
func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error)
```

**Description:** Applies a CCL filter expression to a Parquet file and returns filtered results as a `DataTable`. The filter expression should evaluate to a boolean value for each row.

**Parameters:**

- `ctx`: Context for cancellation
- `path`: Path to the input Parquet file
- `filterExpr`: CCL expression that evaluates to boolean (e.g., `"(A > 100) && (B == 'active')"`)

**Returns:**

- A new `DataTable` containing only rows that satisfy the filter condition, however large the file. When nothing matches, the table has the file's columns and no rows.
- An error when the expression does not compile, names a column the file does not have, cannot be evaluated against a row, or when the file cannot be read — including a read that fails part-way. A read failure is always reported as an error; a partial table is never returned in its place.
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

```go
func ApplyCCL(ctx context.Context, path string, cclScript string, opts ...WriteOptions) error
```

**Description:** Applies CCL expressions directly to a Parquet file in streaming mode, processing data batch by batch to minimize memory usage. The CCL script can contain multiple statements separated by semicolons.

**Parameters:**

- `ctx`: Context for cancellation
- `path`: Path to the Parquet file (will be modified in-place)
- `cclScript`: CCL script containing one or more statements separated by `;` or newlines
- `opts`: Optional, at most one. The compression and row group size to write with, as `Write` uses them; see [WriteOptions](#writeoptions-and-compression). Without it the file keeps its own.

**Returns:**

- `error`: Error when the operation fails.

**Important:**

- The input file **will be overwritten** with the transformed data. The new file goes to a temporary file with a name of its own in the same directory and replaces the original only once it is complete, so a failure leaves the original as it was. An input with no rows leaves it untouched.
- Without `opts`, the file keeps its layout: each column is written with the codec it had, a column the script adds takes the codec of the first column, and row groups are as large as the original's largest one. A 200,000-row Zstd file with one column added used to come back uncompressed in 200 row groups, from 1.7 MB to 9.3 MB.
- With one `WriteOptions`, every column is written with its `Compression` and row groups of its `RowGroupSize`. More than one, or one `Write` would refuse, is an error before the file is read.
- Processing is done in batches to handle large files efficiently. Building a row group larger than a batch holds that row group of the output in memory until it is full. Until every column a statement writes has shown a value, the rows written so far also wait in memory, at most one row group of them, so their types can be settled.
- Each statement sees the file as the statements before it leave it, the way `ExecuteCCL` does on a table: a column an earlier `NEW()` created, or an earlier assignment such as `['A'] = A * 2` replaced, reads its new values. A statement that has to hold whole columns (see [Batches of 1,000 rows](#batches-of-1000-rows)) is computed before anything is written, so an error in it leaves the file as it was.

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

1. **Column Type Consistency**: Each column of the written file holds one type. A column whose values are of several kinds is written as text, as `Write` writes it.

2. **Type Inference**: A column the file had and a statement assigns keeps its type when every value written into it can be held by that type without loss, for the types `ApplyCCL` writes: whole numbers in an `int64` column, numbers a `float64` holds exactly in a `float64` column, text in a text column, booleans, timestamps and byte strings in theirs. Otherwise, and for a column `NEW()` creates, the column takes the type `Write` would give it from all its values. `ApplyCCL` settles the types from the rows it writes first, once every written column has shown a value or a row group is full. When a written column has shown no value by then, or a later value would change a type, it abandons that write, reads the file once more to settle the types from every value, and writes the file again: a run of missing values as long as a row group, such as the first rows of `LAG(A, n)` with `n` at least the row group's size, costs those two extra reads. Every column a statement writes is nullable, so a missing value reads back as `nil`. A column the script does not write keeps the type it had, its time zone included. `ApplyCCL` writes `int64`, `float64`, text, boolean, timestamp and binary columns; a file with a column of another type, such as `int32`, `float32`, a decimal or `Date32`, is an error, and the file is left as it was.

3. **Type Coercion**: Values are converted to the column's type when the file is written, and only where nothing is lost: a whole number computed as `float64` is written into an integer column as that integer, and every value in a text column becomes its text. A fraction is never cut off; it makes the column `float64`.

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
    for dt, err := range parquet.Stream(ctx, "large_data.parquet", parquet.ReadOptions{}, 1000) {
        if err != nil {
            fmt.Println("stream failed:", err)
            return
        }
        numRows, _ := dt.Size()
        fmt.Printf("Batch read, rows: %d\n", numRows)
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
