# Utilities

### 1. `ToFloat64(v interface{}) float64`
This utility converts various numeric types (e.g., `int`, `float32`, `uint`) to `float64`. If the input type is unsupported, it returns `0`.

### 2. `ToFloat64Safe(v interface{}) (float64, bool)`
This function attempts to safely convert a numeric value to `float64`. It returns the converted value and a boolean indicating success or failure.

### 3. `SliceToF64(data []interface{}) []float64`
Converts a slice of `interface{}` values to a slice of `float64`. If conversion fails for any element, a warning is logged, and the function continues processing.

### 4. `ProcessData(input interface{}) ([]interface{}, int)`
Processes input data and returns a slice of `interface{}` and the length of the data. This function supports slices and custom types that implement the `IDataList` interface. For unsupported data types, it returns `nil` and `0`.

### 5. `SqrtRat(x *big.Rat) *big.Rat`
Calculates the square root of a `*big.Rat` (rational number) and returns the result as another `*big.Rat`.

### 6. `PowRat(base *big.Rat, exponent int) *big.Rat`
Computes the power of a `*big.Rat` number raised to a given exponent. This is useful for large or arbitrary precision calculations.

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
