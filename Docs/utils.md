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

#### Parameters

- **data**: `IDataList` type, representing the dependent variable (observations).
- **factor**: `IDataList` type, representing the factor (typically a categorical variable).
- **independents**: `[]IDataList` type, representing multiple independent variables.
- **aggFunc**: `func([]float64) float64` type, a custom aggregation function to handle multiple repeated data entries. If `nil`, the function defaults to returning the first entry.

#### Returns

- Returns an `IDataTable` type containing the wide-format data.

#### Example Usage

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

### 8. `TruncateString(s string, maxLength int) string`

Truncates a string to a specified maximum width, adding an ellipsis (...) at the end if the string is too long. It uses `runewidth` to calculate character width, which is important for handling multi-byte characters correctly.

### 9. `FormatValue(value any) string`

Formats a value of any type for display. It handles various types like numbers, booleans, strings, slices, maps, and structs, providing a clean and readable string representation.

### 10. `ParseColIndex(colName string) int`

Converts an Excel-style column name (e.g., "A", "Z", "AA") to its 0-based integer index.

### 11. `IsNumeric(v any) bool`

Checks if a value is of a numeric type. It supports standard numeric types and also uses reflection to check the underlying kind of an interface value.

### 12. `ColorText(code string, text any) string`

Adds ANSI color codes to text if the environment supports it. This is useful for creating colored console output.

### 13. `isColorSupported() bool`

Detects if the current terminal supports ANSI color codes by checking environment variables like `NO_COLOR` and `TERM`, and also considers the operating system.

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
