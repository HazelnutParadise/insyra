# Utilities

This page covers utility helpers in the main `insyra` package.

### 1. `ToFloat64(v any) float64`

Converts common numeric types to `float64`. Unsupported types return `0`.

### 2. `ToFloat64Safe(v any) (float64, bool)`

Converts numeric values to `float64` and returns a success flag.

### 3. `SliceToF64(data []any) []float64`

Converts a slice of `any` values to a slice of `float64`. Currently only `float64` and `int` are converted; other types become `0`.

### 4. `ProcessData(input any) ([]any, int)`

Normalizes input into a `[]any` and returns its length. Supports slices/arrays, `IDataList`, and pointers to those types. Unsupported types return `nil, 0`.

### 5. `SqrtRat(x *big.Rat) *big.Rat`

Calculates the square root of a `*big.Rat` and returns the result as another `*big.Rat`.

### 6. `PowRat(base *big.Rat, exponent int) *big.Rat`

Computes `base^exponent` for `*big.Rat` values.

### 7. `ConvertLongDataToWide(data, factor IDataList, independents []IDataList, aggFunc func([]float64) float64) IDataTable`

Converts long data into a wide table. When `aggFunc` is nil, the first value is used for duplicates.

```go
data := insyra.NewDataList(5, 15, 25, 35).SetName("Value")
factor := insyra.NewDataList("A", "B", "C", "A").SetName("Factor")
ind1 := insyra.NewDataList(1, 2, 1, 2).SetName("Independent1")
ind2 := insyra.NewDataList(10, 20, 10, 20).SetName("Independent2")

wide := insyra.ConvertLongDataToWide(data, factor, []insyra.IDataList{ind1, ind2}, nil)
wide.Show()
```

### 8. `ParseColIndex(colName string) int`

Converts Excel-style column names (e.g., `A`, `Z`, `AA`) into a 0-based index.

### 9. `IsNumeric(v any) bool`

Returns `true` when `v` is one of Go's standard numeric types.

### 10. `SortTimes(times []time.Time)`

Sorts a slice of `time.Time` in ascending order.

### 11. `Show(label string, object showable, startEnd ...any)`

Displays a labeled preview of any structure that implements `ShowRange` (e.g., `DataTable` or `DataList`). The `startEnd` arguments behave the same as `ShowRange`.

```go
dt := insyra.NewDataTable(
    insyra.NewDataList("Alice", "Bob").SetName("Name"),
    insyra.NewDataList(28, 34).SetName("Age"),
)

insyra.Show("Preview", dt, 1)
```

### 12. `DetectEncoding(filePath string) (string, error)`

Detects the character encoding of a text file. Behavior:

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
