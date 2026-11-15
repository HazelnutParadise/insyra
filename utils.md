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

**Deprecated.** Removed in 0.4. Use `math/big` directly:

```go
f := new(big.Float).SetRat(x)
root, _ := f.Sqrt(f).Rat(nil)
```

**Description:** Returns the square root of `x`, rounded to the precision of a `big.Float` holding `x`. Returns `nil` when `x` is `nil` or negative.

**Parameters:**

- `x`: The value to take the square root of. Type: `*big.Rat`.

**Returns:**

- `*big.Rat`: The square root, or `nil`. Type: `*big.Rat`.

### PowRat

```go
func PowRat(base *big.Rat, exponent int) *big.Rat
```

**Deprecated.** Removed in 0.4. Use `math/big` directly: raise `base.Num()` and `base.Denom()` with `big.Int.Exp` and combine them with `big.Rat.SetFrac`.

**Description:** Returns `base` raised to `exponent` as a new value. A negative exponent gives the reciprocal of the positive power, so `PowRat(big.NewRat(2, 3), -2)` is `9/4`. Any base to the power 0 is 1. Returns `nil` when `base` is `nil`, or when `base` is 0 and `exponent` is negative, since 0 has no reciprocal.

**Parameters:**

- `base`: The value to raise. Type: `*big.Rat`.
- `exponent`: The power, which may be negative. Type: `int`.

**Returns:**

- `*big.Rat`: The power, or `nil`. Type: `*big.Rat`.

### ConvertLongDataToWide

```go
func ConvertLongDataToWide(data, factor IDataList, independents []IDataList, aggFunc func([]float64) float64) IDataTable
```

**Deprecated.** Despite the name, this function does **not** produce a true long-to-wide pivot — it groups rows by `factor` and applies `aggFunc` to every independent column and the observation column independently, yielding one row per factor with one cell per `(factor, column)` pair. The unique values of `factor` are **not** spread into new column headers.

For a real long-to-wide reshape (where unique values of one column become new column headers), use [`(*DataTable).Pivot`](DataTable.md#pivot--unpivot-long--wide-reshape). For the group-and-summarise behaviour this helper actually provides, use [`(*DataTable).GroupBy`](DataTable.md#groupby) followed by `Aggregate` — it is type-safe, surfaces errors via `Err()`, and supports the full `AggregateOp` set.

This function will be removed in a future release.

**Description (legacy):** Converts long data into a wide table. When `aggFunc` is nil, the first value is used for duplicates.

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

### ColIndexToNumber

```go
func ColIndexToNumber(colIndex string) (int, bool)
```

**Description:** Turns an Excel-style column index (`A`, `Z`, `AA`, in either case) into its column number, which counts from 0: `A` is 0 and `AA` is 26. The second return value is `false`, with the number `-1`, for an empty string or any character that is not a letter. `ColNumberToIndex` is the reverse.

**Parameters:**

- `colIndex`: Excel-style column index. Type: `string`.

  **Returns:**

- `int`: Column number, counting from 0. Type: `int`.
- `bool`: Whether `colIndex` is a column index. Type: `bool`.

**Example:**

```go
n, ok := insyra.ColIndexToNumber("AA") // 26, true
```

### ColNumberToIndex

```go
func ColNumberToIndex(colNumber int) (string, bool)
```

**Description:** Turns a column number, which counts from 0, into its Excel-style column index: 0 is `A` and 26 is `AA`. The second return value is `false`, with an empty string, for a negative number. `ColIndexToNumber` is the reverse.

**Parameters:**

- `colNumber`: Column number, counting from 0. Type: `int`.

  **Returns:**

- `string`: Excel-style column index. Type: `string`.
- `bool`: Whether `colNumber` is a column number. Type: `bool`.

**Example:**

```go
index, ok := insyra.ColNumberToIndex(26) // "AA", true
```

> `ParseColIndex` and `CalcColIndex` are the deprecated names of `ColIndexToNumber` and `ColNumberToIndex`. They return the same values and are removed in 0.4.

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

**Deprecated.** Removed in 0.4. Use `slices.SortStableFunc(times, time.Time.Compare)`, which keeps the same stable order.

**Description:** Sorts a slice of `time.Time` in ascending order, in place.

**Parameters:**

- `times`: The times to sort. Type: `[]time.Time`.

**Returns:**

- None.

### Show

```go
func Show(label string, object Showable, startEnd ...any)

type Showable interface {
    ShowRange(startEnd ...any)
}
```

**Description:** Displays a labeled preview of any structure that implements `ShowRange` (e.g., `DataTable`, `DataList` or the `isr` wrappers). The `startEnd` arguments behave the same as `ShowRange`. `Showable` is exported, so a function of your own can take anything `Show` takes:

```go
func preview(items map[string]insyra.Showable) {
    for label, item := range items {
        insyra.Show(label, item, 5)
    }
}
```

**Parameters:**

- `label`: Input value for `label`. Type: `string`.
- `object`: What to show. Type: `Showable`.
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
- Returns `utf-8` if the sample is valid UTF-8. When the sample fills all 8192 bytes, a character cut off by that boundary is ignored; a file shorter than the sample is judged as it is.
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

    "github.com/HazelnutParadise/insyra"
)

func main() {
    num := 42
    f := insyra.ToFloat64(num)
    fmt.Println("Converted:", f)
}
```
