# [ stats ] Package 

Welcome to the **stats** package, which provides efficient functions for calculating statistical measures such as **correlation**, **skewness**, **kurtosis**, **t-tests**, **chi-square tests**, and general **moment calculations**.

## Features

- **Correlation Calculation**: Supports **Pearson**, **Kendall**, and **Spearman** correlation coefficient calculations.
- **T-Test**: Includes **Single Sample**, **Two Sample**, and **Paired** t-tests.
- **Chi-Square Test**: Supports both one-dimensional (1D) and two-dimensional (2D) chi-square tests with optional custom probabilities.
- **Skewness Calculation**: Corresponds directly to **e1071** package's skewness methods.
- **Kurtosis Calculation**: Provides kurtosis calculation that maps directly to **e1071** types.
- **Moment Calculation**: Supports n-th moment calculations for datasets, both central and non-central.
- **OneWayANOVA_WideFormat:** Supports analysis of variance for **wide-format data**.
- **TwoWayANOVA:** Supports analysis of variance for **long-format data**.

## Installation

To install the package, use the following command:

```bash
go get github.com/HazelnutParadise/insyra/stats
```

## Usage

### Correlation

The `Correlation` function calculates the correlation coefficient between two datasets. It supports **Pearson**, **Kendall**, and **Spearman** methods.

```go
import "github.com/HazelnutParadise/insyra/stats"

// Calculate Pearson correlation
result := stats.Correlation(dataListX, dataListY, stats.PearsonCorrelation)
fmt.Println("Pearson correlation:", result)

// Calculate Kendall correlation
result := stats.Correlation(dataListX, dataListY, stats.KendallCorrelation)
fmt.Println("Kendall correlation:", result)

// Calculate Spearman correlation
result := stats.Correlation(dataListX, dataListY, stats.SpearmanCorrelation)
fmt.Println("Spearman correlation:", result)
```

### T-Test

The `TTest` functions allow performing single-sample, two-sample, and paired t-tests.

#### Single Sample T-Test

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.SingleSampleTTest(dataList, 2.5)
fmt.Printf("Single Sample T-Test: t=%.4f, p=%.4f, df=%d\n", result.TValue, result.PValue, result.Df)
```

#### Two Sample T-Test

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.TwoSampleTTest(dataListX, dataListY, true)
fmt.Printf("Two Sample T-Test: t=%.4f, p=%.4f, df=%d\n", result.TValue, result.PValue, result.Df)
```

#### Paired T-Test

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.PairedTTest(dataListX, dataListY)
fmt.Printf("Paired T-Test: t=%.4f, p=%.4f, df=%d\n", result.TValue, result.PValue, result.Df)
```

### Chi-Square Test

The `ChiSquareTest` function supports both **one-dimensional** and **two-dimensional** chi-square tests. You can also specify custom probabilities for the expected values and rescale the probabilities if needed.

#### One-Dimensional Chi-Square Test

```go
import "github.com/HazelnutParadise/insyra/stats"

observed := insyra.NewDataList(20, 15, 25)
p := []float64{1/3, 1/3, 1/3} // Expected probabilities

result := stats.ChiSquareTest(observed, p, true)
fmt.Printf("Chi-Square Value: %.4f, P-Value: %.4f, Degrees of Freedom: %d\n", result.ChiSquare, result.PValue, result.Df)
```

#### Two-Dimensional Chi-Square Test

```go
import "github.com/HazelnutParadise/insyra/stats"

table := insyra.NewDataTable(
    insyra.NewDataList(12, 5, 7),
    insyra.NewDataList(7, 7, 7),
)
result := stats.ChiSquareTest(table, nil, false)
fmt.Printf("Chi-Square Value: %.4f, P-Value: %.4f, Degrees of Freedom: %d\n", result.ChiSquare, result.PValue, result.Df)
```

### Skewness

The `Skewness` function calculates the skewness of a dataset. The method corresponds directly to **e1071** package's `type` options.

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.Skewness(data)
fmt.Println("Skewness:", result)

// Specify the method corresponding to **e1071**'s type
result := stats.Skewness(data, 2) // Corresponds to Type 2 in **e1071**
```

### Kurtosis

The `Kurtosis` function calculates the kurtosis of the dataset. The method corresponds to the **e1071** types.

```go
import "github.com/HazelnutParadise/insyra/stats"

result := stats.Kurtosis(data)
fmt.Println("Kurtosis:", result)

// Specify the method corresponding to **e1071**'s type
result := stats.Kurtosis(data, 3) // Corresponds to Type 3 in **e1071**
```

### Moment Calculation

The `CalculateMoment` function computes the n-th moment of the dataset.

```go
import "github.com/HazelnutParadise/insyra/stats"

moment := stats.CalculateMoment(dataList, 3, true) // Central third moment
fmt.Println("Third moment:", moment)
```

### One-Way ANOVA

The `OneWayANOVA` function computes a One-Way ANOVA for wide-format data using a **DataTable** where each column represents a different group.

```go
import "github.com/HazelnutParadise/insyra/stats"

// Perform One-Way ANOVA on wide-format data
table := insyra.NewDataTable(
    insyra.NewDataList(4, 5, 6),  // Group 1
    insyra.NewDataList(7, 8, 9),  // Group 2
    insyra.NewDataList(1, 2, 3),  // Group 3
)

result := stats.OneWayANOVA(table)
fmt.Printf("SSB: %.2f, SSW: %.2f, F-Value: %.2f, P-Value: %.4f\n", result.SSB, result.SSW, result.FValue, result.PValue)
```

## Method Reference

The methods available for **skewness** and **kurtosis** in this package directly correspond to the `type` options in the **e1071** R package. For further details on the specific calculations and their formulas, please refer to the [e1071 documentation](https://cran.r-project.org/web/packages/e1071/e1071.pdf).
