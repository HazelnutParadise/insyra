# Utilities

### 1. `ToFloat64(v any) float64`

This utility converts various numeric types (e.g., `int`, `float32`, `uint`) to `float64`. If the input type is unsupported, it returns `0`.

### 2. `ToFloat64Safe(v any) (float64, bool)`

This function attempts to safely convert a numeric value to `float64`. It returns the converted value and a boolean indicating success or failure.

### 3. `SliceToF64(data []any) []float64`

Converts a slice of `any` values to a slice of `float64`. If conversion fails for any element, a warning is logged, and the function continues processing.

### 4. `ProcessData(input any) ([]any, int)`

Processes input data and returns a slice of `any` and the length of the data. This function supports slices and custom types that implement the `IDataList` interface. For unsupported data types, it returns `nil` and `0`.

### 5. `SqrtRat(x *big.Rat) *big.Rat`

Calculates the square root of a `*big.Rat` (rational number) and returns the result as another `*big.Rat`.

### 6. `PowRat(base *big.Rat, exponent int) *big.Rat`

Computes the power of a `*big.Rat` number raised to a given exponent. This is useful for large or arbitrary precision calculations.

### 7. `ConvertLongDataToWide(data, factor IDataList, independents []IDataList, aggFunc func([]float64) float64) IDataTable`

Converts long data to wide data.

#### Parameters (Show)

- **data**: `IDataList` type, representing the dependent variable (observations).
- **factor**: `IDataList` type, representing the factor (typically a categorical variable).
- **independents**: `[]IDataList` type, representing multiple independent variables.
- **aggFunc**: `func([]float64) float64` type, a custom aggregation function to handle multiple repeated data entries. If `nil`, the function defaults to returning the first entry.

#### Returns

- Returns an `IDataTable` type containing the wide-format data.

#### Example Usage (Show)

```go
package main

import (
 "fmt"
 "github.com/HazelnutParadise/insyra"
)

func main() {
 // Example data
 data := insyra.NewDataList(5, 15, 25, 35)
 factor := insyra.NewDataList("A", "B", "C", "A").SetName("Factor")
 independent1 := insyra.NewDataList(1, 2, 1, 2).SetName("Independent1")
 independent2 := insyra.NewDataList(10, 20, 10, 20).SetName("Independent2")

 // Convert long data to wide
 wideTable := insyra.ConvertLongDataToWide(data, factor, []insyra.IDataList{independent1, independent2}, nil)
 wideTable.Show()
}
```

This example shows how to convert long-format data to wide-format, including how to handle factors, dependent variables, and multiple independent variables.

### 8. `ParseColIndex(colName string) int`

Converts an Excel-style column name (e.g., "A", "Z", "AA") to its 0-based integer index.

### 9. `IsNumeric(v any) bool`

Checks if a value is of a numeric type. It supports standard numeric types and also uses reflection to check the underlying kind of an interface value.

### 10. `SortTimes(times []time.Time)`

Sorts a slice of `time.Time` in ascending order.

### 11. `Show(label string, object showable, startEnd ...any)`

Displays any Insyra structure that supports ranged output (such as `*insyra.DataTable` and `*insyra.DataList`) with a prefixed label. Internally it delegates to the object's `ShowRange` implementation, so the optional `startEnd` arguments behave exactly the same as calling `ShowRange` directly.

#### Parameters

- **label**: A short string shown before the rendered data. Useful when printing multiple tables or lists.
- **object**: Any value implementing the internal `showable` interface—that is, a type that provides `ShowRange`, including `DataTable`, `DataList`, and other compatible structures.
- **startEnd**: Optional range arguments forwarded to `ShowRange`. Pass nothing to show all rows, a single positive or negative integer to show the first or last _N_ rows, or `[start, end]` pairs (with optional `nil` end) to specify an explicit slice.

#### Example Usage

```go
package main

import (
    "github.com/HazelnutParadise/insyra"
)

func main() {
    dt := insyra.NewDataTable(
        insyra.NewDataList("Alice", "Bob", "Charlie").SetName("Name"),
        insyra.NewDataList(28, 34, 29).SetName("Age"),
    ).SetName("Team Members")

    insyra.Show("Preview", dt, 2)      // Show first 2 rows
    insyra.Show("Latest", dt, -1)      // Show the most recent row
    insyra.Show("Full Table", dt)      // Show entire table
}
```

### 12. `DetectEncoding(filePath string) (string, error)`

**Purpose:** Detect the character encoding of a text file. Supports common encodings such as UTF-8, Big5, GB18030/GBK, and UTF-16 (with BOM).

**Behavior:**

- Reads a sample from the beginning of the file (currently 8192 bytes) to increase detection accuracy.
- Checks for common BOM signatures first:
  - `0xEF 0xBB 0xBF` → `utf-8`
  - `0xFF 0xFE` → `utf-16le`
  - `0xFE 0xFF` → `utf-16be`
- If the sample is valid UTF-8 (`utf8.Valid`), returns `utf-8`.
- Otherwise uses `chardet` to detect encoding and returns the charset in lowercase (e.g., `"big5"`, `"gb-18030"`).
- Returns an error on empty files or when detection fails.

**Return values:**

- (string) detected encoding in lowercase (empty if detection failed)
- (error) non-nil when file open/read or detection fails

**Example usage:**

```go
enc, err := insyra.DetectEncoding("data.csv")
if err != nil {
    // handle detection error or use a fallback (e.g., "utf-8")
}
enc = strings.ToLower(enc)
switch {
case strings.Contains(enc, "big5"):
    // use Big5 decoder
case strings.Contains(enc, "gb"):
    // use GB18030 decoder
case strings.Contains(enc, "utf-16"):
    // use UTF-16 decoder with BOM handling
default:
    // treat as UTF-8
}
```

**Notes:**

- This function is intended for text files; behavior on binary files is undefined and may return incorrect results.
- If you need deterministic behavior, allow callers to force a specific encoding instead of relying on detection.

## Installation

To install **Insyra**, use the following command:

```bash
go get github.com/HazelnutParadise/insyra
```

## Usage

Here is a basic example of how to use Insyra utilities:

```go
package main

import (
    "fmt"
    "math/big"

    "github.com/HazelnutParadise/insyra"
)

func main() {
    // Convert an int to float64
    num := 42
    f := insyra.ToFloat64(num)
    fmt.Println("Converted:", f)

    // Calculate the square root of a big.Rat
    rat := big.NewRat(16, 1)
    sqrtRat := insyra.SqrtRat(rat)
    fmt.Println("Square root:", sqrtRat)
}
```
