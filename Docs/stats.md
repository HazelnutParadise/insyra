# [ stats ] Package 

Welcome to the **stats** package, which provides efficient functions for calculating statistical measures such as **skewness**, **kurtosis**, and general **moment calculations**. The methods used in this package correspond directly to those found in the **e1071** R package.

## Features

- **Skewness Calculation**: Multiple methods for skewness calculation corresponding to those in `e1071`.
- **Kurtosis Calculation**: Provides kurtosis calculation with options that map directly to `e1071`'s types.
- **Moment Calculation**: Supports n-th moment calculations for datasets, both central and non-central.

## Installation

To install the package, use the following command:

```bash
go get github.com/HazelnutParadise/insyra/stats
```

## Usage

### Skewness

The `Skewness` function calculates the skewness of the dataset. The method corresponds directly to the `e1071` package's types.

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.Skewness(data)
fmt.Println("Skewness:", result)

// Specify the method corresponding to `e1071`'s type
result := stats.Skewness(data, 2) // Corresponds to Type 2 in `e1071`
```

### Kurtosis

The `Kurtosis` function calculates the kurtosis of the dataset. The method corresponds to the `e1071` types.

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.Kurtosis(data)
fmt.Println("Kurtosis:", result)

// Specify the method corresponding to `e1071`'s type
result := stats.Kurtosis(data, 3) // Corresponds to Type 3 in `e1071`
```

### Moment Calculation

The `CalculateMoment` function computes the n-th moment of the dataset.

```go
import "github.com/HazelnutParadise/insyra/stats"

moment := stats.CalculateMoment(dataList, 3, true) // Central third moment

fmt.Println("Third moment:", moment)
```

## Method Reference

The methods available for **skewness** and **kurtosis** in this package directly correspond to the `type` options in the **e1071** R package. For further details on the specific calculations and their formulas, please refer to the [e1071 documentation](https://cran.r-project.org/web/packages/e1071/e1071.pdf).
