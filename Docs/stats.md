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
- **Regression Analysis**: Linear, Exponential, Logarithmic, Polynomial regression with confidence intervals
- **F-Tests**: Variance equality, Levene's test, Bartlett's test, regression F-test, nested models
- **Dimensionality Reduction**: Principal Component Analysis (PCA), Factor Analysis
- **Factor Analysis**: Exploratory factor analysis with multiple extraction and rotation methods
- **Matrix Operations**: Diagonal matrix creation and extraction (Diag function)

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

### CorrelationAnalysis

```go
func CorrelationAnalysis(dataTable insyra.IDataTable, method CorrelationMethod) (*insyra.DataTable, *insyra.DataTable, float64, float64, int)
```

**Purpose**: Provides a comprehensive correlation analysis including correlation coefficient matrix, p-value matrix, and overall test (Bartlett's sphericity test).

**Parameters**:

- `dataTable`: Input data table with at least 2 columns
- `method`: Correlation method (see CorrelationMethod below)

**Returns**:

- Correlation coefficient matrix (as DataTable)
- P-value matrix (as DataTable)
- Chi-square value (from Bartlett's sphericity test, only for Pearson correlation)
- P-value of the sphericity test (only for Pearson correlation)
- Degrees of freedom (only for Pearson correlation)

**Example**:

```go
// Calculate correlations with p-values and Bartlett's test
corrMatrix, pMatrix, chiSquare, pValue, df := stats.CorrelationAnalysis(dataTable, stats.PearsonCorrelation)
corrMatrix.Show() // Display the correlation matrix
pMatrix.Show()    // Display the p-value matrix
fmt.Printf("Bartlett's test: χ²=%.4f, p=%.4f, df=%d\n", chiSquare, pValue, df)
corrMatrix.ToCSV("correlation_matrix.csv", true, true, true) // Export to CSV
pMatrix.ToCSV("correlation_matrix_p.csv", true, true, true)  // Export p-values to CSV
```

### CorrelationMatrix

```go
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) (*insyra.DataTable, *insyra.DataTable)
```

**Purpose**: Calculate correlation matrix and corresponding p-value matrix for all columns in a DataTable.

**Parameters**:

- `dataTable`: Input data table with at least 2 columns
- `method`: Correlation method (see CorrelationMethod below)

**Returns**: Two DataTables:

- The first contains the correlation coefficients matrix
- The second contains the p-values matrix
- Both matrices have row and column names matching the original column names

**Example**:

```go
// Calculate correlation matrix with p-values using Pearson correlation
corrMatrix, pMatrix := stats.CorrelationMatrix(dataTable, stats.PearsonCorrelation)
corrMatrix.Show() // Display the correlation matrix
pMatrix.Show()    // Display the p-value matrix
corrMatrix.ToCSV("correlation_matrix.csv", true, true, true) // Export to CSV
pMatrix.ToCSV("correlation_matrix_p.csv", true, true, true)  // Export p-values to CSV
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

### BartlettSphericity

```go
func BartlettSphericity(dataTable insyra.IDataTable) (chiSquare float64, pValue float64, df int)
```

**Purpose**: Performs Bartlett's test of sphericity to assess whether the correlation matrix is significantly different from an identity matrix.

**Parameters**:

- `dataTable`: Input data table with at least 2 columns

**Returns**:

- `chiSquare`: The chi-square statistic
- `pValue`: The p-value of the test
- `df`: The degrees of freedom

**Example**:

```go
// Perform Bartlett's test of sphericity
chiSquare, pValue, df := stats.BartlettSphericity(dataTable)
fmt.Printf("Bartlett's test: χ²=%.4f, p=%.4f, df=%d\n", chiSquare, pValue, df)
// A p-value < 0.05 generally indicates that the correlation matrix is significantly 
// different from an identity matrix, making it suitable for factor analysis or PCA
```

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
func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) *LinearRegressionResult
```

**Purpose**: Performs ordinary least-squares linear regression. Supports both simple (one X) and multiple (multiple X) linear regression.

**Parameters**:

- `dlY`: Dependent variable (IDataList).
- `dlXs`: Independent variable(s) (variadic IDataList). At least one independent variable must be provided. All IDataList inputs must have the same length, and the number of observations must be greater than the number of independent variables.

**Returns**: `*LinearRegressionResult` containing the regression analysis results. Returns `nil` if inputs are invalid (e.g., length mismatch, insufficient observations, singular matrix).

#### LinearRegressionResult

```go
type LinearRegressionResult struct {
    // Legacy fields for simple regression (when only one dlX is provided)
    Slope                  float64 // Regression coefficient β₁ (slope)
    Intercept              float64 // Regression coefficient β₀ (intercept)
    StandardError          float64 // Standard error of the slope (SE(β₁))
    StandardErrorIntercept float64 // Standard error of the intercept (SE(β₀))    TValue                 float64 // t-statistic for the slope (β₁)
    TValueIntercept        float64 // t-statistic for the intercept (β₀)
    PValue                 float64 // Two-tailed p-value for the slope (β₁)
    PValueIntercept        float64 // Two-tailed p-value for the intercept (β₀)

    // Legacy confidence intervals for simple regression compatibility
    ConfidenceIntervalIntercept [2]float64 // 95% confidence interval for intercept [lower, upper]
    ConfidenceIntervalSlope     [2]float64 // 95% confidence interval for slope [lower, upper]
    
    // Extended fields for multiple regression (and also populated for simple regression)
    Coefficients   []float64 // Slice of coefficients: [β₀, β₁, ..., βₚ] (intercept followed by slopes)
    StandardErrors []float64 // Slice of standard errors for each coefficient
    TValues        []float64 // Slice of t-statistics for each coefficient
    PValues        []float64 // Slice of two-tailed p-values for each coefficient

    // Confidence intervals for coefficients (95% by default)
    ConfidenceIntervals [][2]float64 // 95% confidence intervals for each coefficient [lower, upper]

    // Common fields for both simple and multiple regression
    Residuals        []float64 // Residuals (yᵢ − ŷᵢ)
    RSquared         float64   // Coefficient of determination (R²)
    AdjustedRSquared float64   // Adjusted R²
}
```

**Fields in LinearRegressionResult**:

- **Legacy Fields (for simple regression, where `len(dlXs) == 1`)**:
  - `Slope`: The slope of the regression line (β₁).
  - `Intercept`: The y-intercept of the regression line (β₀).
  - `StandardError`: The standard error of the slope coefficient.
  - `StandardErrorIntercept`: The standard error of the intercept coefficient.
  - `TValue`: The t-statistic for the slope, used to test its significance.  - `TValueIntercept`: The t-statistic for the intercept, used to test its significance.
  - `PValue`: The p-value associated with the t-statistic for the slope.
  - `PValueIntercept`: The p-value associated with the t-statistic for the intercept.
  - `ConfidenceIntervalIntercept`: The 95% confidence interval for the intercept `[lower_bound, upper_bound]`.
  - `ConfidenceIntervalSlope`: The 95% confidence interval for the slope `[lower_bound, upper_bound]`.

- **Extended Fields (for multiple regression, also available for simple regression)**:
  - `Coefficients`: A slice containing all model coefficients. The first element (`Coefficients[0]`) is the intercept (β₀), and subsequent elements (`Coefficients[1:]`) are the coefficients for the independent variables (β₁, β₂, ..., βₚ).
  - `StandardErrors`: A slice of standard errors corresponding to each coefficient in `Coefficients`.  - `TValues`: A slice of t-statistics corresponding to each coefficient.
  - `PValues`: A slice of p-values corresponding to each t-statistic.
  - `ConfidenceIntervals`: A slice of 95% confidence intervals for each coefficient. Each element is a `[2]float64` array containing `[lower_bound, upper_bound]`.

- **Common Fields**:
  - `Residuals`: A slice of the differences between the observed and predicted values for the dependent variable.
  - `RSquared`: The proportion of the variance in the dependent variable that is predictable from the independent variable(s).
  - `AdjustedRSquared`: R-squared adjusted for the number of predictors in the model.

**Example (Simple Linear Regression)**:

```go
y := insyra.NewDataList([]float64{1, 2, 3, 4, 5})
x1 := insyra.NewDataList([]float64{2, 4, 5, 4, 5})

result := stats.LinearRegression(y, x1)
if result != nil {
    fmt.Printf("Simple Linear Regression:\n")
    fmt.Printf("  Intercept (β₀): %.4f (p=%.4f)\n", result.Intercept, result.PValueIntercept)
    fmt.Printf("  Slope (β₁ for x1): %.4f (p=%.4f)\n", result.Slope, result.PValue)
    fmt.Printf("  R-squared: %.4f\n", result.RSquared)
    fmt.Printf("  Adjusted R-squared: %.4f\n", result.AdjustedRSquared)
    
    // Display 95% confidence intervals using clear field names
    fmt.Printf("  95%% CI for Intercept: [%.4f, %.4f]\n", 
        result.ConfidenceIntervalIntercept[0], result.ConfidenceIntervalIntercept[1])
    fmt.Printf("  95%% CI for Slope: [%.4f, %.4f]\n", 
        result.ConfidenceIntervalSlope[0], result.ConfidenceIntervalSlope[1])
    
    // Alternative: using array format (useful for multiple regression)
    // fmt.Printf("  95%% CI for Intercept: [%.4f, %.4f]\n", 
    //     result.ConfidenceIntervals[0][0], result.ConfidenceIntervals[0][1])
    // fmt.Printf("  95%% CI for Slope: [%.4f, %.4f]\n", 
    //     result.ConfidenceIntervals[1][0], result.ConfidenceIntervals[1][1])
}
```

**Example (Multiple Linear Regression)**:

```go
y := insyra.NewDataList([]float64{15, 25, 30, 35, 40, 50})
x1 := insyra.NewDataList([]float64{2, 3, 4, 5, 6, 7})
x2 := insyra.NewDataList([]float64{1, 2, 2, 3, 3, 4})

result := stats.LinearRegression(y, x1, x2)
if result != nil {
    fmt.Printf("\nMultiple Linear Regression:\n")
    fmt.Printf("  Intercept (β₀): %.4f (p=%.4f)\n", result.Coefficients[0], result.PValues[0])
    for i := 1; i < len(result.Coefficients); i++ {
        fmt.Printf("  Slope (β%d for x%d): %.4f (p=%.4f)\n", i, i, result.Coefficients[i], result.PValues[i])
    }
    fmt.Printf("  R-squared: %.4f\n", result.RSquared)
    fmt.Printf("  Adjusted R-squared: %.4f\n", result.AdjustedRSquared)
    
    // Display 95% confidence intervals for all coefficients
    fmt.Printf("  95%% Confidence Intervals:\n")
    fmt.Printf("    Intercept: [%.4f, %.4f]\n", 
        result.ConfidenceIntervals[0][0], result.ConfidenceIntervals[0][1])
    for i := 1; i < len(result.ConfidenceIntervals); i++ {
        fmt.Printf("    β%d: [%.4f, %.4f]\n", i,
            result.ConfidenceIntervals[i][0], result.ConfidenceIntervals[i][1])
    }
}
```

### PolynomialRegression

```go
func PolynomialRegression(dlY insyra.IDataList, dlX insyra.IDataList, degree int) *PolynomialRegressionResult
```

**Purpose**: Perform polynomial regression analysis (y = a₀ + a₁x + a₂x² + ... + aₙxⁿ).

**Parameters**:

- `dlY`: Dependent variable data
- `dlX`: Independent variable data
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
    
    // Confidence intervals for coefficients (95% by default)
    ConfidenceIntervals [][2]float64 // 95% confidence intervals for each coefficient [lower, upper]
}
```

**Example**:

```go
// Perform cubic polynomial regression: y = a₀ + a₁x + a₂x² + a₃x³
result := stats.PolynomialRegression(yData, xData, 3) // Corrected parameter order
fmt.Printf("Equation: y = %.4f + %.4f·x + %.4f·x² + %.4f·x³\\n", \r
    result.Coefficients[0], result.Coefficients[1], \r
    result.Coefficients[2], result.Coefficients[3])
fmt.Printf("R² = %.4f\\n", result.RSquared)
```

### ExponentialRegression

```go
func ExponentialRegression(dlY insyra.IDataList, dlX insyra.IDataList) *ExponentialRegressionResult
```

**Purpose**: Perform exponential regression analysis (y = a·e^(b·x)).

**Parameters**:

- `dlY`: Dependent variable data (must contain positive values)
- `dlX`: Independent variable data

**Returns**: Result containing coefficients (a, b) and model evaluation metrics.

#### ExponentialRegressionResult

```go
type ExponentialRegressionResult struct {
    Intercept              float64   // coefficient a in y = a·e^(b·x)
    Slope                  float64   // coefficient b in y = a·e^(b·x)
    Residuals              []float64 // yᵢ − ŷᵢ
    RSquared               float64   // coefficient of determination
    AdjustedRSquared       float64   // adjusted R²
    StandardErrorIntercept float64   // standard error of coefficient a
    StandardErrorSlope     float64   // standard error of coefficient b
    TValueIntercept        float64   // t statistic for coefficient a
    TValueSlope            float64   // t statistic for coefficient b
    PValueIntercept        float64   // p-value for coefficient a
    PValueSlope            float64   // p-value for coefficient b
    
    // Confidence intervals for coefficients (95% by default)
    ConfidenceIntervalIntercept [2]float64 // 95% confidence interval for intercept [lower, upper]
    ConfidenceIntervalSlope     [2]float64 // 95% confidence interval for slope [lower, upper]
}
```

**Example**:

```go
// Perform exponential regression: y = a·e^(b·x)
result := stats.ExponentialRegression(yData, xData) // Corrected parameter order
if result != nil {
    fmt.Printf("Equation: y = %.4f · e^(%.4f·x)\n", result.Intercept, result.Slope)
    fmt.Printf("R² = %.4f\n", result.RSquared)
    
    // Display 95% confidence intervals
    fmt.Printf("95%% CI for Intercept (a): [%.4f, %.4f]\n", 
        result.ConfidenceIntervalIntercept[0], result.ConfidenceIntervalIntercept[1])
    fmt.Printf("95%% CI for Slope (b): [%.4f, %.4f]\n", 
        result.ConfidenceIntervalSlope[0], result.ConfidenceIntervalSlope[1])
}
```

### LogarithmicRegression

```go
func LogarithmicRegression(dlY insyra.IDataList, dlX insyra.IDataList) *LogarithmicRegressionResult
```

**Purpose**: Perform logarithmic regression analysis (y = a + b·ln(x)).

**Parameters**:

- `dlY`: Dependent variable data
- `dlX`: Independent variable data (must contain positive values)

**Returns**: Result containing coefficients (a, b) and model evaluation metrics.

#### LogarithmicRegressionResult

```go
type LogarithmicRegressionResult struct {
    Intercept              float64   // intercept coefficient in y = a + b·ln(x)
    Slope                  float64   // slope coefficient in y = a + b·ln(x)
    Residuals              []float64 // yᵢ − ŷᵢ
    RSquared               float64   // coefficient of determination
    AdjustedRSquared       float64   // adjusted R²
    StandardErrorIntercept float64   // standard error of coefficient a
    StandardErrorSlope     float64   // standard error of coefficient b
    TValueIntercept        float64   // t statistic for coefficient a
    TValueSlope            float64   // t statistic for coefficient b
    PValueIntercept        float64   // p-value for coefficient a
    PValueSlope            float64   // p-value for coefficient b
    
    // Confidence intervals for coefficients (95% by default)
    ConfidenceIntervalIntercept [2]float64 // 95% confidence interval for intercept [lower, upper]
    ConfidenceIntervalSlope     [2]float64 // 95% confidence interval for slope [lower, upper]
}
```

**Example**:

```go
// Perform logarithmic regression: y = a + b·ln(x)
result := stats.LogarithmicRegression(yData, xData) // Corrected parameter order
if result != nil {
    fmt.Printf("Equation: y = %.4f + %.4f·ln(x)\n", result.Intercept, result.Slope)
    fmt.Printf("R² = %.4f\n", result.RSquared)
    
    // Display 95% confidence intervals
    fmt.Printf("95%% CI for Intercept (a): [%.4f, %.4f]\n", 
        result.ConfidenceIntervalIntercept[0], result.ConfidenceIntervalIntercept[1])
    fmt.Printf("95%% CI for Slope (b): [%.4f, %.4f]\n", 
        result.ConfidenceIntervalSlope[0], result.ConfidenceIntervalSlope[1])
}
```

---

## Factor Analysis

The stats package provides comprehensive factor analysis functionality for dimensionality reduction and latent variable identification. Factor analysis helps identify underlying factors that explain the correlations among observed variables.

### Model Framework

Factor analysis assumes the observed correlation (or covariance) matrix can be decomposed into shared factor structure plus unique variance:

$$
R \approx F F' + U^2
$$

Where:

- $R$ — observed correlation matrix (identity on the diagonal when standardized)
- $F$ — factor loading matrix capturing relationships between variables and common factors
- $U^2$ — diagonal matrix of uniquenesses (specific variance not explained by common factors)

Communalities are the diagonal elements of $F F'$, representing shared variance for each variable. The goal of extraction algorithms is to find loadings $F$ and uniquenesses $U^2$ that best approximate $R$ under different optimality criteria.

### Extraction Methods

The implementation mirrors key `psych::fa` extraction behaviors while exposing the choice through `FactorAnalysisOptions.Extraction`. The currently selectable values are `pca`, `paf`, `ml`, and `minres`; the remaining subsections document the underlying theory used for compatibility and roadmap planning.

#### Principal Axis Factoring (PAF / `paf`)

- **Initialization**: Communalities start from squared multiple correlations (SMC) or a user-provided constant.
- **Iteration**:
    1. Replace the diagonal of $R$ with current communalities.
    2. Perform an eigen decomposition to obtain updated loadings $F$.
    3. Update communalities via $\operatorname{diag}(F F')$.
    4. Repeat until the maximum change in communalities is below `Tol` or `MaxIter` iterations are reached.
- **Strengths**: Stable and fast; often converges even when ML struggles.

#### Minimum Residual (MINRES / `minres`)

- **Objective**: Minimize the off-diagonal squared residuals between the observed and model-implied correlations:

    $$
    \min_{F, U^2} \sum_{i<j} \bigl(r_{ij} - \hat{r}_{ij}\bigr)^2,
    $$

    where $\hat{r}_{ij}$ are elements of $F F' + U^2$.
- **Optimization**: Uses gradient-based search (mirroring `psych::fa` ≥ 2017 derivatives) with diagonal adjustments to maintain feasible uniquenesses.
- **Traits**: Default extraction (matches `psych::fa`). Robust when communalities are moderate to high.

#### Maximum Likelihood (ML / `ml`)

- **Objective**: Minimize the Gaussian log-likelihood discrepancy between the model-implied and observed matrices:

    $$
    f = \log\bigl(\operatorname{trace}((F F' + U^2)^{-1} R)\bigr) - \log\bigl(\lvert (F F' + U^2)^{-1} R \rvert\bigr) - p,
    $$

    with $p$ as the number of observed variables.
- **Algorithm**: Newton-style steps iterate on communalities; convergence enables chi-square goodness-of-fit testing:

    $$
    \chi^2 = (n_{\text{obs}} - 1 - \tfrac{2p + 5}{6} - \tfrac{2f}{3}) \cdot f,
    $$

    where $f$ is the minimized value and $n_{\text{obs}}$ is the number of observations.
- **Notes**: Requires positive definite $F F' + U^2$ and benefits from multivariate normal data.

#### Weighted / Generalized Least Squares (WLS & GLS)

- **Weighting**:
  - WLS weights residuals by $1 / \operatorname{diag}(R^{-1})$.
  - GLS weights residuals by the full $R^{-1}$.
- **Effect**: Variables with lower communalities receive higher weight, emphasizing residual reduction for poorly explained variables.
- **Use Case**: Useful when measurement error varies substantially across variables.

#### Minimum Rank Factor Analysis (MRFA)

- **Goal**: Find the lowest-rank approximation of $R$ that preserves positive semidefiniteness of the residual matrix.
- **Behavior**: Ensures the resulting residual covariance remains a valid correlation matrix, avoiding negative eigenvalues.
- **When to Choose**: Specialized method for guaranteeing admissible residual structures.

#### Alpha Factoring (`alpha`)

- **Process**: Adjusts $R$ by subtracting communalities, rescales to the original metric, and extracts factors maximizing Cronbach's alpha.
- **Origin**: Based on Kaiser & Coffey (1965).
- **Strength**: Produces factors optimized for reliability of sum scores.

### Rotation Families

Rotation is configured through `FactorAnalysisOptions.Rotation`. Orthogonal methods keep factors uncorrelated ($\Phi = I$), while oblique methods estimate $\Phi$.

- **Orthogonal**: Varimax (default), Quartimax.
- **Oblique**: Promax (power `Kappa`), Oblimin (parameter `Delta`).

For oblique solutions, the structure matrix $S$ and pattern matrix $P$ obey:

$$
S = P \Phi,
$$

linking loadings ($P$) to factor correlations ($\Phi$).

### Convergence Control

- `FactorAnalysisOptions.MaxIter` (default `50`, matching R's psych::fa) caps iterations for iterative extractions (PAF, MINRES, ML, MRFA, Alpha).
- Diagnostics: Outputs include `FactorAnalysisResult.Converged` and `Iterations` to track termination status.

### Factor Scoring

Factor scores follow Thurstone-style regression by default. Given standardized data matrix $X$ and structure matrix $S$:

$$
F_s = X W, \quad W = R^{-1} S.
$$

Alternative weighting schemes (Bartlett, Anderson–Rubin, ten Berge) are accessible via `FactorAnalysisOptions.Scoring`. Scoring reuses preprocessing parameters (standardization, missing policy) captured inside the `FactorModel`.

### FactorAnalysis

```go
func FactorAnalysis(dt insyra.IDataTable, opt FactorAnalysisOptions) *FactorModel
```

**Purpose**: Perform factor analysis on a dataset to identify underlying latent factors.

**Parameters**:

- `dt`: Input data table with numeric columns (variables × observations)
- `opt`: Factor analysis options (see FactorAnalysisOptions below)

**Returns**: `*FactorModel` containing the factor analysis results and model for scoring new data. Returns `nil` if analysis fails, with warnings logged via `LogWarning()`.

#### FactorAnalysisOptions

```go
type FactorAnalysisOptions struct {
    Preprocess FactorPreprocessOptions
    Count      FactorCountSpec
    Extraction FactorExtractionMethod
    Rotation   FactorRotationOptions
    Scoring    FactorScoreMethod
    MaxIter    int     // Maximum iterations for iterative methods (default: 50)
    MinErr     float64 // Min error for convergence (default: 0.001)
}
```

#### FactorPreprocessOptions

```go
type FactorPreprocessOptions struct {
    Standardize bool    // Whether to standardize variables (default: true)
    Missing     string  // Missing data handling: "listwise", "pairwise", "mean" (default: "listwise")
}
```

#### FactorCountSpec

```go
type FactorCountSpec struct {
    Method         FactorCountMethod  // Method to determine number of factors
    FixedK         int                // Number of factors for CountFixed
    EigenThreshold float64            // Eigenvalue threshold for CountKaiser (default: 1.0)
    MaxFactors     int                // Maximum number of factors to extract
}
```

#### FactorCountMethod

```go
type FactorCountMethod string
const (
    FactorCountFixed  FactorCountMethod = "fixed"
    FactorCountKaiser FactorCountMethod = "kaiser"
)
```

#### FactorExtractionMethod

```go
type FactorExtractionMethod string
const (
    FactorExtractionPCA    FactorExtractionMethod = "pca"
    FactorExtractionPAF      FactorExtractionMethod = "paf"
    FactorExtractionML       FactorExtractionMethod = "ml"
    FactorExtractionMINRES   FactorExtractionMethod = "minres"
)
```

#### FactorRotationOptions

```go
type FactorRotationOptions struct {
    Method FactorRotationMethod
    Kappa  float64 // For Promax (default: 4)
    Delta  float64 // For Oblimin (default: 0)
}
```

#### FactorRotationMethod

```go
type FactorRotationMethod string
const (
    FactorRotationNone      FactorRotationMethod = "none"
    FactorRotationVarimax   FactorRotationMethod = "varimax"
    FactorRotationQuartimax FactorRotationMethod = "quartimax"
    FactorRotationQuartimin FactorRotationMethod = "quartimin"
    FactorRotationOblimin   FactorRotationMethod = "oblimin"
    FactorRotationGeominT   FactorRotationMethod = "geominT"
    FactorRotationBentlerT  FactorRotationMethod = "bentlerT"
    FactorRotationSimplimax FactorRotationMethod = "simplimax"
    FactorRotationGeominQ   FactorRotationMethod = "geominQ"
    FactorRotationBentlerQ  FactorRotationMethod = "bentlerQ"
    FactorRotationPromax    FactorRotationMethod = "promax"
)
```

#### FactorScoreMethod

```go
type FactorScoreMethod string
const (
    FactorScoreNone          FactorScoreMethod = "none"
    FactorScoreRegression    FactorScoreMethod = "regression"
    FactorScoreBartlett      FactorScoreMethod = "bartlett"
    FactorScoreAndersonRubin FactorScoreMethod = "anderson-rubin"
)
```

#### FactorAnalysisResult

```go
type FactorAnalysisResult struct {
    Loadings             insyra.IDataTable // Loading matrix (variables × factors)
    Structure            insyra.IDataTable // Structure matrix (variables × factors)
    Uniquenesses         insyra.IDataTable // Uniqueness vector (p × 1)
    Communalities        insyra.IDataTable // Communality vector (p × 1)
    Phi                  insyra.IDataTable // Factor correlation matrix (m × m), nil for orthogonal
    RotationMatrix       insyra.IDataTable // Rotation matrix (m × m), nil if no rotation
    Eigenvalues          insyra.IDataTable // Eigenvalues vector (p × 1)
    ExplainedProportion  insyra.IDataTable // Proportion explained by each factor (m × 1)
    CumulativeProportion insyra.IDataTable // Cumulative proportion explained (m × 1)
    Scores               insyra.IDataTable // Factor scores (n × m), nil if not computed

    Converged  bool
    Iterations int
    CountUsed  int
    Messages   []string
}
```

**DataTable Naming Convention**:

- **Loadings**: Column names are factor names (Factor1, Factor2, ...), row names are variable names
- **Structure**: Column names are factor names (Factor1, Factor2, ...), row names are variable names
- **Uniquenesses**: Single column named "Uniqueness", row names are variable names
- **Communalities**: Single column named "Communality", row names are variable names
- **Eigenvalues**: Single column named "Eigenvalue", row names are factor names
- **ExplainedProportion**: Single column named "Explained Proportion", row names are factor names
- **CumulativeProportion**: Single column named "Cumulative Proportion", row names are factor names
- **Scores**: Column names are factor names, row names are observation indices

#### Show Method

```go
func (r *FactorAnalysisResult) Show(startEndRange ...any)
```

**Purpose**: Display all factor analysis results in a formatted manner.

**Parameters**:

- `startEndRange`: Optional range parameters for displaying DataTables (passed to Show method)

**Example**:

```go
// Display all factor analysis results
model.Show()
```

#### FactorModel

```go
type FactorModel struct {
    FactorAnalysisResult

    // Internal fields for scoring new data
    scoreMethod FactorScoreMethod
    extraction  FactorExtractionMethod
    rotation    FactorRotationMethod
    means       []float64
    sds         []float64
}
```

**Example**:

```go
// Perform factor analysis with default options
model := stats.FactorAnalysis(dataTable, stats.DefaultFactorAnalysisOptions())
if model == nil {
    log.Fatal("Factor analysis failed")
}

// Display results
model.Loadings.Show()        // Factor loadings
model.Communalities.Show()   // Communalities
model.Eigenvalues.Show()     // Eigenvalues

// Export results
model.Loadings.ToCSV("factor_loadings.csv", true, true, true)
```

### FactorScores

```go
func (m *FactorModel) FactorScores(dt insyra.IDataTable, method *FactorScoreMethod) (insyra.IDataTable, error)
```

**Purpose**: Compute factor scores for new data using a fitted factor analysis model.

**Parameters**:

- `dt`: New data table with the same variables as the original analysis
- `method`: Optional scoring method (uses model's default if nil)

**Returns**: Factor scores as a DataTable (observations × factors).

**Example**:

```go
// Compute factor scores for new data
scores, err := model.FactorScores(newData, nil)
if err != nil {
    log.Fatal(err)
}
scores.Show()
scores.ToCSV("factor_scores.csv", true, true, true)
```

### ScreePlotData

```go
func ScreePlotData(dt insyra.IDataTable, standardize bool) (eigenDT insyra.IDataTable, cumDT insyra.IDataTable, err error)
```

**Purpose**: Returns scree plot data (eigenvalues and cumulative proportion) for determining the number of factors to extract.

**Parameters**:

- `dt`: Input data table
- `standardize`: Whether to standardize variables before analysis

**Returns**:

- `eigenDT`: DataTable containing eigenvalues in descending order
- `cumDT`: DataTable containing cumulative proportions of explained variance
- `err`: Error if analysis fails

**Example**:

```go
// Get scree plot data for factor analysis
eigenvalues, cumulative, err := stats.ScreePlotData(dataTable, true)
if err != nil {
    log.Fatal(err)
}
eigenvalues.Show() // Display eigenvalues
cumulative.Show()  // Display cumulative proportions
```

### DefaultFactorAnalysisOptions

```go
func DefaultFactorAnalysisOptions() FactorAnalysisOptions
```

**Purpose**: Returns default factor analysis options aligned with R's `psych::fa` defaults.

**Returns**: Default FactorAnalysisOptions with the following defaults:

- **Extraction**: `minres` (Minimum Residual)
- **Rotation**: `oblimin` (Oblique rotation with delta=0)
- **Scoring**: `regression` (Regression-based factor scores)
- **Preprocessing**: Standardize=true, Missing="listwise"
- **Factor Count**: Kaiser criterion (eigenvalues > 1.0)
- **MaxIter**: 50
- **MinErr**: 0.001

---

## Matrix Operations

### Diag

```go
func Diag(x any, dims ...int) any
```

**Purpose**: Create diagonal matrices or extract diagonal elements from matrices, mimicking R's `diag()` function.

**Parameters**:

- `x`: Input value of various types:
  - `*mat.Dense`: Extract diagonal elements as `[]float64`
  - `[]float64`: Create diagonal matrix from slice
  - `int` or `float64`: Create identity matrix of specified size
  - `nil`: Create identity matrix (default 1x1)
- `dims`: Optional dimensions (0, 1, or 2 values):
  - No dims: Use default sizing based on input
  - 1 dim: Set nrow = ncol = dim[0]
  - 2 dims: Set nrow = dim[0], ncol = dim[1]

**Returns**:

- When extracting: `[]float64` containing diagonal elements
- When creating: `*mat.Dense` diagonal or identity matrix**Examples**:

```go
// Extract diagonal from matrix
matrix := mat.NewDense(3, 3, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9})
diagonal := Diag(matrix) // Returns []float64{1, 5, 9}

// Create diagonal matrix from slice
values := []float64{1, 2, 3}
diagMatrix := Diag(values) // Returns 3x3 diagonal matrix

// Create identity matrix
identity := Diag(3) // Returns 3x3 identity matrix

// Create rectangular identity matrix
rectIdentity := Diag(nil, 2, 3) // Returns 2x3 matrix with diagonal 1s
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

### Confidence Intervals for Regression Analysis

All regression functions (Linear, Polynomial, Exponential, and Logarithmic) now provide 95% confidence intervals for their coefficients:

- **Linear and Polynomial Regression**: Returns `ConfidenceIntervals [][2]float64` containing confidence intervals for all coefficients (intercept and slopes).
- **Exponential and Logarithmic Regression**: Returns separate `ConfidenceIntervalIntercept [2]float64` and `ConfidenceIntervalSlope [2]float64` fields.

The confidence intervals are calculated using the t-distribution with appropriate degrees of freedom:

```text
CI = coefficient ± t_(α/2, df) × standard_error
```

Where:

- `t_(α/2, df)` is the critical value from the t-distribution
- `df` is the degrees of freedom (sample_size - number_of_parameters)
- `α = 0.05` for 95% confidence intervals

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

### Factor Analysis Best Practices

#### Data Preparation

- **Standardization**: Standardize variables before factor analysis unless they are already on comparable scales
- **Missing Data**: Use "listwise" deletion for missing data unless the dataset is small
- **Sample Size**: Aim for at least 5-10 observations per variable, preferably more
- **Variable Selection**: Include only variables that are theoretically related to the construct

#### Factor Extraction

- **MINRES**: Default choice (psych::fa), stable and works well without strict distributional assumptions
- **PAF**: Preferred when communalities are low, often yields interpretable factor structures
- **ML**: Maximum likelihood, requires multivariate normality, provides fit statistics
- **PCA**: Deterministic extraction based on eigen decomposition; useful for exploratory variance explanation

#### Determining Number of Factors

- **Kaiser Criterion**: Eigenvalues > 1.0 (default, conservative)
- **Scree Plot**: Look for "elbow" in the plot of eigenvalues (manual inspection)
- **Fixed**: When theory specifies the number of factors

#### Factor Rotation

- **Varimax**: Orthogonal rotation, maximizes variance of squared loadings (default)
- **Promax**: Oblique rotation, allows correlated factors, more realistic
- **None**: Keep original unrotated solution for interpretation

#### Interpretation

- **Loadings > 0.3**: Generally considered significant
- **Loadings > 0.5**: Strong relationship
- **Communality > 0.5**: Variable well-explained by factors
- **Cross-loadings**: Variables loading highly on multiple factors may need removal

#### Validation

- **Factor Scores**: Use for further analysis or clustering
- **Model Fit**: Check convergence and proportion of explained variance
- **Reproducibility**: Validate on holdout samples when possible
