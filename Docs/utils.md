# Utilities

This page covers utility helpers in the main `insyra` package.

### ToFloat64

```go
func ToFloat64(v any) float64
```

**Description:** Converts common numeric types to `float64`. Unsupported types return `0`, which cannot be told apart from a real zero; use `ToFloat64Safe` when that matters.

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

**Description:** Converts a slice of `any` values to a slice of `float64`. Every numeric value, of any int, uint or float width, is converted; any other value becomes `0`.

**Parameters:**

- `data`: Input data values. Type: `[]any`.

**Returns:**

- `[]float64`: Result slice. Type: `[]float64`.

### ProcessData

```go
func ProcessData(input any) ([]any, error)
```

**Description:** Reads `input` as a `[]any`: a slice or an array element by element, an `IDataList` by its values, and a pointer to a slice or an array by what it points to. An empty slice gives an empty, non-nil result. Any other type, a `nil` input and a nil pointer, a nil `*DataList` included, return an error and a nil slice.

**Parameters:**

- `input`: The values to read. Type: `any`.

**Returns:**

- `[]any`: The values, in order. Type: `[]any`.
- `error`: Why `input` could not be read, or nil. Type: `error`.

```go
values, err := insyra.ProcessData([]float64{0.2, 0.3, 0.5})
if err != nil {
    // input was not a slice, an array or an IDataList
}
fmt.Println(len(values)) // 3
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

### IsNumeric

```go
func IsNumeric(v any) bool
```

**Description:** Returns `true` when `v` is one of Go's standard numeric types.

**Parameters:**

- `v`: Input value for `v`. Type: `any`.

**Returns:**

- `bool`: Computed value. Type: `bool`.

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
- Falls back to `chardet` for other encodings, and uses its answer only when it is clear: a confidence of at least 50 out of 100, with no other charset scoring the same.
- Returns an error for empty files, and an error wrapping `ErrEncodingAmbiguous` when `chardet`'s answer is not clear. The message lists the likeliest charsets with their scores; name the encoding to read such a file.
- Returns `utf-8`, with a warning, when `chardet` names no charset at all.

```go
enc, err := insyra.DetectEncoding("data.csv")
if errors.Is(err, insyra.ErrEncodingAmbiguous) {
    // the file does not show its encoding; ask for it or use a known one
} else if err != nil {
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

    values, err := insyra.ProcessData([]int{1, 2, 3})
    if err != nil {
        fmt.Println("cannot read:", err)
        return
    }
    fmt.Println("Values:", values)
}
```
