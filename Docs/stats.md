# stats Package Documentation

This document describes all public APIs in the `stats` package, designed for AI/automated applications to directly understand each function, type, parameter, and return value.

---

## Installation

```bash
go get github.com/HazelnutParadise/insyra/stats
```

---

## Overview

The stats package provides comprehensive statistical analysis functions:

- **Correlation Analysis**: Pearson, Kendall, Spearman correlation coefficients, correlation matrices
- **Hypothesis Testing**: t-tests (single, two-sample, paired), z-tests, chi-square tests
- **Distribution Analysis**: Skewness, Kurtosis, n-th moments
- **Analysis of Variance**: One-way, Two-way, Repeated measures ANOVA
- **Regression Analysis**: Linear, Exponential, Logarithmic, Polynomial regression
- **F-Tests**: Variance equality, Levene's test, Bartlett's test, regression F-test, nested models
- **Dimensionality Reduction**: Principal Component Analysis (PCA)

---

## Core Types

### Common Result Structure

```go
type testResultBase struct {
    Statistic   float64           // Test statistic value
    PValue      float64           // P-value
    DF          *float64          // Degrees of freedom (nil if not applicable)
    CI          *[2]float64       // Confidence interval (nil if not calculated)
    EffectSizes []EffectSizeEntry // Effect size measures
}

type EffectSizeEntry struct {
    Type  string  // "cohen_d", "hedges_g", etc.
    Value float64 // Effect size value
}
```

### Alternative Hypothesis

```go
type AlternativeHypothesis string
const (
    TwoSided AlternativeHypothesis = "two-sided"
    Greater  AlternativeHypothesis = "greater"
    Less     AlternativeHypothesis = "less"
)
```

---

## Correlation Analysis

### CorrelationMatrix

```go
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) *insyra.DataTable
```

**Purpose**: Calculate correlation matrix for all columns in a DataTable.

**Parameters**:

- `dataTable`: Input data table with at least 2 columns
- `method`: Correlation method (see CorrelationMethod below)

**Returns**: A new DataTable containing the correlation matrix with row and column names matching the original column names.

**Example**:

```go
// Calculate correlation matrix using Pearson correlation
corMatrix := stats.CorrelationMatrix(dataTable, stats.PearsonCorrelation)
corMatrix.Show() // Display the correlation matrix
corMatrix.ToCSV("correlation_matrix.csv", true, true, true) // Export to CSV
```

### Correlation

```go
func Correlation(dlX, dlY insyra.IDataList, method CorrelationMethod) *CorrelationResult
```

**Purpose**: Calculate correlation coefficient between two datasets.

**Parameters**:

- `dlX, dlY`: Input data lists (must have same length, minimum 2 elements)
- `method`: Correlation method (see CorrelationMethod below)

**Returns**: `*CorrelationResult` containing correlation coefficient and statistical significance.

#### CorrelationMethod

```go
type CorrelationMethod int
const (
    PearsonCorrelation  CorrelationMethod = iota // Linear correlation
    KendallCorrelation                           // Rank-based correlation (robust)
    SpearmanCorrelation                          // Monotonic correlation
)
```

#### CorrelationResult

```go
type CorrelationResult struct {
    testResultBase // Statistic = correlation coefficient, PValue = significance
}
```

**Example**:

```go
result := stats.Correlation(dataX, dataY, stats.PearsonCorrelation)
fmt.Printf("Correlation: %.4f, P-value: %.4f\n", result.Statistic, result.PValue)
```

### Covariance

```go
func Covariance(dlX, dlY insyra.IDataList) float64
```

**Purpose**: Calculate sample covariance between two datasets.

---

## T-Tests

### SingleSampleTTest

```go
func SingleSampleTTest(data insyra.IDataList, mu float64, confidenceLevel float64) *TTestResult
```

**Purpose**: Test if sample mean differs from hypothesized population mean.

**Parameters**:

- `data`: Sample data (minimum 2 elements)
- `mu`: Hypothesized population mean
- `confidenceLevel`: Confidence level (0 < confidenceLevel < 1, default 0.95)

### TwoSampleTTest

```go
func TwoSampleTTest(data1, data2 insyra.IDataList, equalVariance bool, confidenceLevel ...float64) *TTestResult
```

**Purpose**: Compare means of two independent samples.

**Parameters**:

- `data1, data2`: Two independent samples
- `equalVariance`: true for pooled variance, false for Welch's t-test
- `confidenceLevel`: Optional confidence level (default 0.95)

### PairedTTest

```go
func PairedTTest(data1, data2 insyra.IDataList, confidenceLevel ...float64) *TTestResult
```

**Purpose**: Compare means of paired/dependent samples.

**Parameters**:

- `data1, data2`: Paired samples (must have same length)
- `confidenceLevel`: Optional confidence level (default 0.95)

#### TTestResult

```go
type TTestResult struct {
    testResultBase
    Mean     *float64 // Mean of first group
    Mean2    *float64 // Mean of second group (nil for single sample)
    MeanDiff *float64 // Mean difference (paired t-test only)
    N        int      // Sample size of first group
    N2       *int     // Sample size of second group (nil for single/paired)
}
```

**Example**:

```go
// Single sample t-test
result := stats.SingleSampleTTest(data, 100.0, 0.95)
fmt.Printf("t=%.4f, p=%.4f, df=%.0f\n", result.Statistic, result.PValue, *result.DF)

// Two sample t-test
result := stats.TwoSampleTTest(group1, group2, true, 0.95)
fmt.Printf("t=%.4f, p=%.4f\n", result.Statistic, result.PValue)

// Paired t-test
result := stats.PairedTTest(before, after, 0.95)
fmt.Printf("t=%.4f, p=%.4f, mean diff=%.4f\n", result.Statistic, result.PValue, *result.MeanDiff)
```

---

## Z-Tests

### SingleSampleZTest

```go
func SingleSampleZTest(data insyra.IDataList, mu float64, sigma float64, alternative AlternativeHypothesis, confidenceLevel float64) *ZTestResult
```

**Purpose**: Test sample mean against population mean when population standard deviation is known.

**Parameters**:

- `data`: Sample data
- `mu`: Hypothesized population mean
- `sigma`: Known population standard deviation (must be > 0)
- `alternative`: Type of alternative hypothesis
- `confidenceLevel`: Confidence level (0 < confidenceLevel < 1)

#### ZTestResult

```go
type ZTestResult struct {
    testResultBase
    Mean  float64  // Sample mean
    Mean2 *float64 // Second sample mean (nil for single sample)
    N     int      // Sample size
    N2    *int     // Second sample size (nil for single sample)
}
```

**Example**:

```go
result := stats.SingleSampleZTest(data, 100.0, 15.0, stats.TwoSided, 0.95)
fmt.Printf("z=%.4f, p=%.4f\n", result.Statistic, result.PValue)
```

---

## Chi-Square Tests

### ChiSquareGoodnessOfFit

```go
func ChiSquareGoodnessOfFit(input insyra.IDataList, p []float64, rescaleP bool) *ChiSquareTestResult
```

**Purpose**: Test if observed frequencies match expected frequencies.

**Parameters**:

- `input`: Observed frequencies
- `p`: Expected probabilities (nil for equal probabilities)
- `rescaleP`: Whether to rescale probabilities to sum to 1

### ChiSquareIndependenceTest

```go
func ChiSquareIndependenceTest(rowData, colData insyra.IDataList) *ChiSquareTestResult
```

**Purpose**: Test independence between two categorical variables.

**Parameters**:

- `rowData, colData`: Categorical data lists

#### ChiSquareTestResult

```go
type ChiSquareTestResult struct {
    testResultBase // Statistic = chi-square statistic
}
```

**Example**:

```go
// Goodness of fit test
observed := insyra.NewDataList(20, 15, 25)
p := []float64{1.0/3, 1.0/3, 1.0/3}
result := stats.ChiSquareGoodnessOfFit(observed, p, true)
fmt.Printf("Chi-square=%.4f, p=%.4f, df=%.0f\n", result.Statistic, result.PValue, *result.DF)

// Independence test
result := stats.ChiSquareIndependenceTest(rowData, colData)
fmt.Printf("Chi-square=%.4f, p=%.4f\n", result.Statistic, result.PValue)
```

---

## Distribution Analysis

### Skewness

```go
func Skewness(sample any, method ...SkewnessMethod) float64
```

**Purpose**: Calculate skewness (asymmetry) of a distribution.

**Parameters**:

- `sample`: Data (any type convertible to float64 slice)
- `method`: Optional skewness calculation method (default: SkewnessG1)

#### SkewnessMethod

```go
type SkewnessMethod int
const (
    SkewnessG1           SkewnessMethod = iota + 1 // Type 1: G1 (default)
    SkewnessAdjusted                               // Type 2: Adjusted Fisher-Pearson
    SkewnessBiasAdjusted                           // Type 3: Bias-adjusted
)
```

**Returns**: Skewness value (float64). NaN if data is invalid.

### Kurtosis

```go
func Kurtosis(data any, method ...KurtosisMethod) float64
```

**Purpose**: Calculate kurtosis (tail heaviness) of a distribution.

**Parameters**:

- `data`: Data (any type convertible to float64 slice)
- `method`: Optional kurtosis calculation method (default: KurtosisG2)

#### KurtosisMethod

```go
type KurtosisMethod int
const (
    KurtosisG2           KurtosisMethod = iota + 1 // Type 1: g2 (default)
    KurtosisAdjusted                               // Type 2: adjusted Fisher kurtosis
    KurtosisBiasAdjusted                           // Type 3: bias-adjusted
)
```

**Returns**: Kurtosis value (float64). NaN if data is invalid.

### CalculateMoment

```go
func CalculateMoment(dl insyra.IDataList, n int, central bool) float64
```

**Purpose**: Calculate n-th moment of a dataset.

**Parameters**:

- `dl`: Input data list
- `n`: Moment order (positive integer)
- `central`: true for central moments, false for raw moments

**Returns**: n-th moment value (float64). NaN if calculation fails.

**Example**:

```go
// Calculate skewness
skew := stats.Skewness(data, stats.SkewnessG1)
fmt.Printf("Skewness: %.4f\n", skew)

// Calculate kurtosis
kurt := stats.Kurtosis(data, stats.KurtosisG2)
fmt.Printf("Kurtosis: %.4f\n", kurt)

// Calculate 3rd central moment
moment3 := stats.CalculateMoment(dataList, 3, true)
fmt.Printf("3rd central moment: %.4f\n", moment3)
```

---

## Analysis of Variance (ANOVA)

### OneWayANOVA

```go
func OneWayANOVA(groups ...insyra.IDataList) *OneWayANOVAResult
```

**Purpose**: Compare means across multiple independent groups.

**Parameters**:

- `groups`: Variable number of data groups (minimum 2 groups)

### TwoWayANOVA

```go
func TwoWayANOVA(factorALevels, factorBLevels int, cells ...insyra.IDataList) *TwoWayANOVAResult
```

**Purpose**: Analyze effects of two factors and their interaction.

**Parameters**:

- `factorALevels, factorBLevels`: Number of levels for each factor
- `cells`: Data for each factor combination

#### ANOVA Result Types

```go
type OneWayANOVAResult struct {
    Factor  ANOVAResultComponent
    Within  ANOVAResultComponent
    TotalSS float64
}

type TwoWayANOVAResult struct {
    FactorA     ANOVAResultComponent
    FactorB     ANOVAResultComponent
    Interaction ANOVAResultComponent
    Within      ANOVAResultComponent
    TotalSS     float64
}

type ANOVAResultComponent struct {
    SumOfSquares float64
    DF           int
    F            float64
    P            float64
    EtaSquared   float64
}
```

**Example**:

```go
// One-way ANOVA
group1 := insyra.NewDataList(4, 5, 6)
group2 := insyra.NewDataList(7, 8, 9)
group3 := insyra.NewDataList(1, 2, 3)
result := stats.OneWayANOVA(group1, group2, group3)
fmt.Printf("F=%.4f, p=%.4f\n", result.Factor.F, result.Factor.P)

// Two-way ANOVA
cells := []insyra.IDataList{
    insyra.NewDataList(1, 2, 3),  // A1B1
    insyra.NewDataList(4, 5, 6),  // A1B2
    insyra.NewDataList(7, 8, 9),  // A2B1
    insyra.NewDataList(10, 11, 12), // A2B2
}
result := stats.TwoWayANOVA(2, 2, cells...)
fmt.Printf("Factor A F=%.4f, p=%.4f\n", result.FactorA.F, result.FactorA.P)
```

---

## F-Tests

### FTestForVarianceEquality

```go
func FTestForVarianceEquality(data1, data2 insyra.IDataList) *FTestResult
```

**Purpose**: Test equality of variances between two groups.

### LeveneTest

```go
func LeveneTest(groups []insyra.IDataList) *FTestResult
```

**Purpose**: Test equality of variances across multiple groups (robust).

### BartlettTest

```go
func BartlettTest(groups []insyra.IDataList) *FTestResult
```

**Purpose**: Test equality of variances across multiple groups (assumes normality).

### FTestForRegression

```go
func FTestForRegression(ssr, sse float64, df1, df2 int) *FTestResult
```

**Purpose**: Test overall significance of regression model.

**Parameters**:

- `ssr`: Regression sum of squares
- `sse`: Error sum of squares
- `df1, df2`: Degrees of freedom

### FTestForNestedModels

```go
func FTestForNestedModels(rssReduced, rssFull float64, dfReduced, dfFull int) *FTestResult
```

**Purpose**: Compare nested regression models.

**Parameters**:

- `rssReduced, rssFull`: Residual sum of squares for each model
- `dfReduced, dfFull`: Degrees of freedom for each model

#### FTestResult

```go
type FTestResult struct {
    testResultBase
    DF2 float64 // Second degrees of freedom
}
```

**Example**:

```go
// Test variance equality
result := stats.FTestForVarianceEquality(group1, group2)
fmt.Printf("F=%.4f, p=%.4f\n", result.Statistic, result.PValue)

// Levene's test
groups := []insyra.IDataList{group1, group2, group3}
result := stats.LeveneTest(groups)
fmt.Printf("Levene F=%.4f, p=%.4f\n", result.Statistic, result.PValue)

// Regression F-test
result := stats.FTestForRegression(1200.0, 800.0, 3, 20)
fmt.Printf("Regression F=%.4f, p=%.4f\n", result.Statistic, result.PValue)
```

---

## Principal Component Analysis (PCA)

### PCA

```go
func PCA(dataTable insyra.IDataTable, nComponents ...int) *PCAResult
```

**Purpose**: Perform principal component analysis to reduce dimensionality of data.

**Parameters**:

- `dataTable`: Input data table with observations in rows and variables in columns
- `nComponents`: Optional number of components to extract (default: all components)

**Returns**: A `PCAResult` structure containing the principal components, eigenvalues, and explained variance.

#### PCAResult

```go
type PCAResult struct {
    Components        insyra.IDataTable // Principal component loadings as DataTable
    Eigenvalues       []float64         // Eigenvalues corresponding to components
    ExplainedVariance []float64         // Percentage of variance explained by each component
}
```

**Example**:

```go
// Perform PCA with 2 components
result := stats.PCA(dataTable, 2)

// Access components and explained variance
components := result.Components
fmt.Printf("Explained variance: %.2f%%\n", result.ExplainedVariance[0])
```

## Regression Analysis

### LinearRegression

```go
func LinearRegression(dlX, dlY insyra.IDataList) *LinearRegressionResult
```

**Purpose**: Perform simple linear regression analysis (y = a + bx).

**Parameters**:

- `dlX`: Independent variable data
- `dlY`: Dependent variable data

**Returns**: Result containing regression coefficients, statistical significance, and model evaluation metrics.

#### LinearRegressionResult

```go
type LinearRegressionResult struct {
    Slope                  float64   // Regression coefficient β₁
    Intercept              float64   // Regression coefficient β₀
    Residuals              []float64 // Residual values (yᵢ − ŷᵢ)
    RSquared               float64   // Coefficient of determination
    AdjustedRSquared       float64   // Adjusted R²
    StandardError          float64   // SE(β₁) - slope standard error
    StandardErrorIntercept float64   // SE(β₀) - intercept standard error
    TValue                 float64   // t statistic for β₁
    TValueIntercept        float64   // t statistic for β₀
    PValue                 float64   // p-value for β₁
    PValueIntercept        float64   // p-value for β₀
}
```

**Example**:

```go
result := stats.LinearRegression(xData, yData)
fmt.Printf("y = %.4fx + %.4f\n", result.Slope, result.Intercept)
fmt.Printf("R² = %.4f, p = %.4f\n", result.RSquared, result.PValue)
```

### PolynomialRegression

```go
func PolynomialRegression(dlX, dlY insyra.IDataList, degree int) *PolynomialRegressionResult
```

**Purpose**: Perform polynomial regression analysis (y = a₀ + a₁x + a₂x² + ... + aₙxⁿ).

**Parameters**:

- `dlX`: Independent variable data
- `dlY`: Dependent variable data
- `degree`: Degree of the polynomial (≥ 1)

**Returns**: Result containing polynomial coefficients and model evaluation metrics.

#### PolynomialRegressionResult

```go
type PolynomialRegressionResult struct {
    Coefficients     []float64 // Polynomial coefficients [a₀, a₁, a₂, ...]
    Degree           int       // Degree of polynomial
    Residuals        []float64 // Residual values (yᵢ − ŷᵢ)
    RSquared         float64   // Coefficient of determination
    AdjustedRSquared float64   // Adjusted R²
    StandardErrors   []float64 // Standard errors for each coefficient
    TValues          []float64 // t statistics for each coefficient
    PValues          []float64 // p-values for each coefficient
}
```

**Example**:

```go
// Perform cubic polynomial regression: y = a₀ + a₁x + a₂x² + a₃x³
result := stats.PolynomialRegression(xData, yData, 3)
fmt.Printf("Equation: y = %.4f + %.4f·x + %.4f·x² + %.4f·x³\n", 
    result.Coefficients[0], result.Coefficients[1], 
    result.Coefficients[2], result.Coefficients[3])
fmt.Printf("R² = %.4f\n", result.RSquared)
```

### ExponentialRegression

```go
func ExponentialRegression(dlX, dlY insyra.IDataList) *ExponentialRegressionResult
```

**Purpose**: Perform exponential regression analysis (y = a·e^(b·x)).

**Parameters**:

- `dlX`: Independent variable data
- `dlY`: Dependent variable data (all values must be positive)

**Returns**: Result containing exponential model coefficients and evaluation metrics.

#### ExponentialRegressionResult

```go
type ExponentialRegressionResult struct {
    Intercept              float64   // Coefficient a in y = a·e^(b·x)
    Slope                  float64   // Coefficient b in y = a·e^(b·x)
    Residuals              []float64 // Residual values (yᵢ − ŷᵢ)
    RSquared               float64   // Coefficient of determination
    AdjustedRSquared       float64   // Adjusted R²
    StandardErrorIntercept float64   // Standard error of coefficient a
    StandardErrorSlope     float64   // Standard error of coefficient b
    TValueIntercept        float64   // t statistic for coefficient a
    TValueSlope            float64   // t statistic for coefficient b
    PValueIntercept        float64   // p-value for coefficient a
    PValueSlope            float64   // p-value for coefficient b
}
```

**Example**:

```go
result := stats.ExponentialRegression(xData, yData)
fmt.Printf("y = %.4f·e^(%.4f·x)\n", result.Intercept, result.Slope)
fmt.Printf("R² = %.4f\n", result.RSquared)
```

### LogarithmicRegression

```go
func LogarithmicRegression(dlX, dlY insyra.IDataList) *LogarithmicRegressionResult
```

**Purpose**: Perform logarithmic regression analysis (y = a + b·ln(x)).

**Parameters**:

- `dlX`: Independent variable data (all values must be positive)
- `dlY`: Dependent variable data

**Returns**: Result containing logarithmic model coefficients and evaluation metrics.

#### LogarithmicRegressionResult

```go
type LogarithmicRegressionResult struct {
    Intercept              float64   // Intercept coefficient in y = a + b·ln(x)
    Slope                  float64   // Slope coefficient in y = a + b·ln(x)
    Residuals              []float64 // Residual values (yᵢ − ŷᵢ)
    RSquared               float64   // Coefficient of determination
    AdjustedRSquared       float64   // Adjusted R²
    StandardErrorIntercept float64   // Standard error of coefficient a
    StandardErrorSlope     float64   // Standard error of coefficient b
    TValueIntercept        float64   // t statistic for coefficient a
    TValueSlope            float64   // t statistic for coefficient b
    PValueIntercept        float64   // p-value for coefficient a
    PValueSlope            float64   // p-value for coefficient b
}
```

**Example**:

```go
result := stats.LogarithmicRegression(xData, yData)
fmt.Printf("y = %.4f + %.4f·ln(x)\n", result.Intercept, result.Slope)
fmt.Printf("R² = %.4f\n", result.RSquared)
```

---

## Method Reference

### Skewness and Kurtosis Methods

The methods available for skewness and kurtosis calculations correspond directly to the `type` options in the R package `e1071`:

- **Type 1** (G1/g2): Default methods using sample moments
- **Type 2** (Adjusted): Adjusted Fisher-Pearson estimators  
- **Type 3** (Bias-adjusted): Bias-corrected versions

For detailed mathematical formulas, refer to the [e1071 documentation](https://cran.r-project.org/web/packages/e1071/e1071.pdf).

### Confidence Levels

Most functions accept optional confidence levels. If not specified or invalid (outside 0-1 range), the default confidence level of 0.95 (95%) is used.

### Data Input Types

Functions accept various data input types:

- `insyra.IDataList`: Primary interface for data lists
- `any`: Raw data that can be converted to float64 slices
- `[]float64`: Direct float64 slices where applicable

### Error Handling

Functions return `nil` or `NaN` values when:

- Input data is empty or invalid
- Sample sizes are too small for the test
- Mathematical requirements are not met
- Invalid parameter combinations are provided

All error conditions are logged via `insyra.LogWarning()` for debugging purposes.
