# Utilities

This page covers utility helpers in the main `insyra` package.

### ToFloat64

```go
func ToFloat64(v any) float64
```

**Description:** Converts common numeric types to `float64`. Unsupported types return `0`.

**Parameters:**

- `v`: Input value for `v`. Type: `any`.

**Returns:**

- `float64`: Computed value. Type: `float64`.

### ToFloat64Safe

```go
func ToFloat64Safe(v any) (float64, bool)
```

**Description:** Converts numeric values to `float64` and returns a success flag.

**Parameters:**

- `v`: Input value for `v`. Type: `any`.

**Returns:**

- `float64`: Computed value. Type: `float64`.
- `bool`: Computed value. Type: `bool`.

### SliceToF64

```go
func SliceToF64(data []any) []float64
```

**Description:** Converts a slice of `any` values to a slice of `float64`. Currently only `float64` and `int` are converted; other types become `0`.

**Parameters:**

- `data`: Input data values. Type: `[]any`.

**Returns:**

- `[]float64`: Result slice. Type: `[]float64`.

### ProcessData

```go
func ProcessData(input any) ([]any, int)
```

**Description:** Normalizes input into a `[]any` and returns its length. Supports slices/arrays, `IDataList`, and pointers to those types. Unsupported types return `nil, 0`.

**Parameters:**

- `input`: Input value for `input`. Type: `any`.

**Returns:**

- `[]any`: Result slice. Type: `[]any`.
- `int`: Computed value. Type: `int`.

### SqrtRat

```go
func SqrtRat(x *big.Rat) *big.Rat
```

**Description:** Calculates the square root of a `*big.Rat` and returns the result as another `*big.Rat`.

**Parameters:**

- `x`: Numeric parameter value. Type: `*big.Rat`.

**Returns:**

- `*big.Rat`: Return value. Type: `*big.Rat`.

### PowRat

```go
func PowRat(base *big.Rat, exponent int) *big.Rat
```

**Description:** Computes `base^exponent` for `*big.Rat` values.

**Parameters:**

- `base`: Input value for `base`. Type: `*big.Rat`.
- `exponent`: Input value for `exponent`. Type: `int`.

**Returns:**

- `*big.Rat`: Return value. Type: `*big.Rat`.

### ConvertLongDataToWide

```go
func ConvertLongDataToWide(data, factor IDataList, independents []IDataList, aggFunc func([]float64) float64) IDataTable
```

**Description:** Converts long data into a wide table. When `aggFunc` is nil, the first value is used for duplicates.

**Parameters:**

- `data`: Input data values. Type: `IDataList`.
- `factor`: Input value for `factor`. Type: `IDataList`.
- `independents`: Input value for `independents`. Type: `[]IDataList`.
- `aggFunc`: Input value for `aggFunc`. Type: `func([]float64) float64`.

**Returns:**

- `IDataTable`: Resulting data table. Type: `IDataTable`.

```go
data := insyra.NewDataList(5, 15, 25, 35).SetName("Value")
factor := insyra.NewDataList("A", "B", "C", "A").SetName("Factor")
ind1 := insyra.NewDataList(1, 2, 1, 2).SetName("Independent1")
ind2 := insyra.NewDataList(10, 20, 10, 20).SetName("Independent2")

wide := insyra.ConvertLongDataToWide(data, factor, []insyra.IDataList{ind1, ind2}, nil)
wide.Show()
```

### ParseColIndex

```go
func ParseColIndex(colName string) (int, bool)
```

**Description:** Converts Excel-style column names (e.g., `A`, `Z`, `AA`) into a 0-based index. The second return value is a boolean `ok` indicating whether parsing succeeded (false for invalid column names).

**Parameters:**

- `colName`: Name value to use. Type: `string`.

**Returns:**

- `int`: Computed value (0-based index). Type: `int`.
- `bool`: Success flag (`ok`) indicating whether parsing succeeded. Type: `bool`.

### IsNumeric

```go
func IsNumeric(v any) bool
```

**Description:** Returns `true` when `v` is one of Go's standard numeric types.

**Parameters:**

- `v`: Input value for `v`. Type: `any`.

**Returns:**

- `bool`: Computed value. Type: `bool`.

### SortTimes

```go
func SortTimes(times []time.Time)
```

**Description:** Sorts a slice of `time.Time` in ascending order.

**Parameters:**

- `times`: Input value for `times`. Type: `[]time.Time`.

**Returns:**

- None.

### Show

```go
func Show(label string, object showable, startEnd ...any)
```

**Description:** Displays a labeled preview of any structure that implements `ShowRange` (e.g., `DataTable` or `DataList`). The `startEnd` arguments behave the same as `ShowRange`.

**Parameters:**

- `label`: Input value for `label`. Type: `string`.
- `object`: Input value for `object`. Type: `showable`.
- `startEnd`: Input value for `startEnd`. Type: `...any`.

**Returns:**

- None.

```go
dt := insyra.NewDataTable(
    insyra.NewDataList("Alice", "Bob").SetName("Name"),
    insyra.NewDataList(28, 34).SetName("Age"),
)

insyra.Show("Preview", dt, 1)
```

### DetectEncoding

```go
func DetectEncoding(filePath string) (string, error)
```

**Description:** Detects the character encoding of a text file. Behavior:

**Parameters:**

- `filePath`: File path to use. Type: `string`.

**Returns:**

- `string`: Computed value. Type: `string`.
- `error`: Error when the operation fails. Type: `error`.

- Reads a 8192-byte sample for detection.
- Checks BOM markers (`utf-8`, `utf-16le`, `utf-16be`).
- Returns `utf-8` if the sample is valid UTF-8.
- Falls back to `chardet` for other encodings.
- Returns an error for empty files or failed detection.

```go
enc, err := insyra.DetectEncoding("data.csv")
if err != nil {
    // handle detection error
}
```

## Installation

```bash
go get github.com/HazelnutParadise/insyra
```

## Usage

```go
package main

import (
    "fmt"
    "math/big"

    "github.com/HazelnutParadise/insyra"
)

func main() {
    num := 42
    f := insyra.ToFloat64(num)
    fmt.Println("Converted:", f)

    rat := big.NewRat(16, 1)
    sqrtRat := insyra.SqrtRat(rat)
    fmt.Println("Square root:", sqrtRat)
}
```
