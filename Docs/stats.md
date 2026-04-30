# [ stats ] Package

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
- **Dimensionality Reduction**: Principal Component Analysis (PCA)
- **Instance-Based Prediction**: K-nearest neighbors (KNN) classification and regression
- **Clustering Analysis**: K-means, hierarchical agglomerative clustering, DBSCAN, silhouette analysis
- **Matrix Operations**: Diagonal matrix creation and extraction (Diag function)

Most functions expect numeric data in `DataList`/`DataTable` and return `error` when inputs are invalid or computation fails. Always handle `err` at call sites.

For clustering APIs, Insyra uses an R-oriented result shape and cross-language verification policy:

- R is the authoritative semantic reference for clustering behavior and output structure
- Python baselines are used as a verification companion, not as the source of truth
- v1 clustering distance support is Euclidean-only
- `KMeansOptions.Seed` is an Insyra reproducibility extension for deterministic validation
- Clustering computations are implemented in pure Go; R and Python are used only for parity validation in tests

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

### Correlation Analysis

```go
func CorrelationAnalysis(dataTable insyra.IDataTable, method CorrelationMethod) (*insyra.DataTable, *insyra.DataTable, float64, float64, int, error)
```

**Description:** Provides a comprehensive correlation analysis including correlation coefficient matrix, p-value matrix, and overall test (Bartlett's sphericity test).

**Parameters:**

- `dataTable`: Input data table with at least 2 columns
- `method`: Correlation method (see CorrelationMethod below)

**Returns:**

- Correlation coefficient matrix (as DataTable)
- P-value matrix (as DataTable)
- Chi-square value (from Bartlett's sphericity test, only for Pearson correlation)
- P-value of the sphericity test (only for Pearson correlation)
- Degrees of freedom (only for Pearson correlation)

**Example**:

```go
// Calculate correlations with p-values and Bartlett's test
corrMatrix, pMatrix, chiSquare, pValue, df, err := stats.CorrelationAnalysis(dataTable, stats.PearsonCorrelation)
if err != nil {
    log.Fatal(err)
}
corrMatrix.Show() // Display the correlation matrix
pMatrix.Show()    // Display the p-value matrix
fmt.Printf("Bartlett's test: chi-square=%.4f, p=%.4f, df=%d\n", chiSquare, pValue, df)
corrMatrix.ToCSV("correlation_matrix.csv", true, true, true) // Export to CSV
pMatrix.ToCSV("correlation_matrix_p.csv", true, true, true)  // Export p-values to CSV
```

### Correlation Matrix

```go
func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) (*insyra.DataTable, *insyra.DataTable, error)
```

**Description:** Calculate correlation matrix and corresponding p-value matrix for all columns in a DataTable.

**Parameters:**

- `dataTable`: Input data table with at least 2 columns
- `method`: Correlation method (see CorrelationMethod below)

**Returns:**
Two DataTables:

- The first contains the correlation coefficients matrix
- The second contains the p-values matrix
- Both matrices have row and column names matching the original column names

**Example**:

```go
// Calculate correlation matrix with p-values using Pearson correlation
corrMatrix, pMatrix, err := stats.CorrelationMatrix(dataTable, stats.PearsonCorrelation)
if err != nil {
    log.Fatal(err)
}
corrMatrix.Show() // Display the correlation matrix
pMatrix.Show()    // Display the p-value matrix
corrMatrix.ToCSV("correlation_matrix.csv", true, true, true) // Export to CSV
pMatrix.ToCSV("correlation_matrix_p.csv", true, true, true)  // Export p-values to CSV
```

### Correlation

```go
func Correlation(dlX, dlY insyra.IDataList, method CorrelationMethod) (*CorrelationResult, error)
```

**Description:** Calculate correlation coefficient between two datasets.

**Parameters:**

- `dlX, dlY`: Input data lists (must have same length, minimum 2 elements)
- `method`: Correlation method (see CorrelationMethod below)

**Returns:**

- `*CorrelationResult`: Return value.

#### Correlation Method

```go
type CorrelationMethod int
const (
    PearsonCorrelation  CorrelationMethod = iota // Linear correlation
    KendallCorrelation                           // Rank-based correlation (robust)
    SpearmanCorrelation                          // Monotonic correlation
)
```

#### Correlation Result

```go
type CorrelationResult struct {
    testResultBase // Statistic = correlation coefficient, PValue = significance
}
```

**Example**:

```go
result, err := stats.Correlation(dataX, dataY, stats.PearsonCorrelation)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Correlation: %.4f, P-value: %.4f\n", result.Statistic, result.PValue)
```

### Covariance

```go
func Covariance(dlX, dlY insyra.IDataList) (float64, error)
```

**Description:** Calculate sample covariance between two datasets.

**Parameters:**

- `dlX`: Input value for `dlX`. Type: `insyra.IDataList`.
- `dlY`: Input value for `dlY`. Type: `insyra.IDataList`.

**Returns:**

- `float64`: Return value.

### Bartlett Sphericity

```go
func BartlettSphericity(dataTable insyra.IDataTable) (chiSquare float64, pValue float64, df int, err error)
```

**Description:** Performs Bartlett's test of sphericity to assess whether the correlation matrix is significantly different from an identity matrix.

**Parameters:**

- `dataTable`: Input data table with at least 2 columns

**Returns:**

- `chiSquare`: The chi-square statistic
- `pValue`: The p-value of the test
- `df`: The degrees of freedom

**Example**:

```go
// Perform Bartlett's test of sphericity
chiSquare, pValue, df, err := stats.BartlettSphericity(dataTable)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Bartlett's test: chi-square=%.4f, p=%.4f, df=%d\n", chiSquare, pValue, df)
// A p-value < 0.05 generally indicates that the correlation matrix is significantly
// different from an identity matrix, making it suitable for factor analysis
```

---

## T-Tests

### Single Sample T-Test

```go
func SingleSampleTTest(data insyra.IDataList, mu float64, confidenceLevel ...float64) (*TTestResult, error)
```

**Description:** Test if sample mean differs from hypothesized population mean.

**Parameters:**

- `data`: Sample data (minimum 2 elements)
- `mu`: Hypothesized population mean
- `confidenceLevel`: Confidence level (0 < confidenceLevel < 1, default 0.95)

**Returns:**

- `*TTestResult`: Return value.

### Two Sample T-Test

```go
func TwoSampleTTest(data1, data2 insyra.IDataList, equalVariance bool, confidenceLevel ...float64) (*TTestResult, error)
```

**Description:** Compare means of two independent samples.

**Parameters:**

- `data1, data2`: Two independent samples
- `equalVariance`: true for pooled variance, false for Welch's t-test
- `confidenceLevel`: Optional confidence level (default 0.95)

**Returns:**

- `*TTestResult`: Return value.

### Paired T-Test

```go
func PairedTTest(data1, data2 insyra.IDataList, confidenceLevel ...float64) (*TTestResult, error)
```

**Description:** Compare means of paired/dependent samples.

**Parameters:**

- `data1, data2`: Paired samples (must have same length)
- `confidenceLevel`: Optional confidence level (default 0.95)

**Returns:**

- `*TTestResult`: Return value.

#### T-Test Result

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
result, err := stats.SingleSampleTTest(data, 100.0, 0.95)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("t=%.4f, p=%.4f, df=%.0f\n", result.Statistic, result.PValue, *result.DF)

// Two sample t-test
result, err = stats.TwoSampleTTest(group1, group2, true, 0.95)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("t=%.4f, p=%.4f\n", result.Statistic, result.PValue)

// Paired t-test
result, err = stats.PairedTTest(before, after, 0.95)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("t=%.4f, p=%.4f, mean diff=%.4f\n", result.Statistic, result.PValue, *result.MeanDiff)
```

---

## Z-Tests

### Single Sample Z-Test

```go
func SingleSampleZTest(data insyra.IDataList, mu float64, sigma float64, alternative AlternativeHypothesis, confidenceLevel float64) (*ZTestResult, error)
```

**Description:** Test sample mean against population mean when population standard deviation is known.

**Parameters:**

- `data`: Sample data
- `mu`: Hypothesized population mean
- `sigma`: Known population standard deviation (must be > 0)
- `alternative`: Type of alternative hypothesis
- `confidenceLevel`: Confidence level (0 < confidenceLevel < 1)

**Returns:**

- `*ZTestResult`: Return value.

#### Z-Test Result

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
result, err := stats.SingleSampleZTest(data, 100.0, 15.0, stats.TwoSided, 0.95)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("z=%.4f, p=%.4f\n", result.Statistic, result.PValue)
```

---

## Chi-Square Tests

### Chi-Square Goodness of Fit

```go
func ChiSquareGoodnessOfFit(input insyra.IDataList, p []float64, rescaleP bool) (*ChiSquareTestResult, error)
```

**Description:** Test if observed categorical data matches expected distribution.

**Parameters:**

- `input`: Categorical data (e.g., ["A", "B", "A"])
- `p`: Expected probabilities (nil for uniform distribution)
- `rescaleP`: Whether to rescale probabilities to sum to 1

**Returns:**

- `*ChiSquareTestResult`: Return value.

### Chi-Square Independence Test

```go
func ChiSquareIndependenceTest(rowData, colData insyra.IDataList) (*ChiSquareTestResult, error)
```

**Description:** Test independence between two categorical variables.

**Parameters:**

- `rowData, colData`: Categorical data lists

**Returns:**

- `*ChiSquareTestResult`: Return value.

#### Chi-Square Test Result

```go
type ChiSquareTestResult struct {
    testResultBase           // Statistic = chi-square statistic
    ContingencyTable *insyra.DataTable // Contingency table with observed and expected values
}
```

The `ContingencyTable` contains the observed frequencies and expected frequencies for each cell in the contingency table. For goodness of fit tests, it shows observed vs expected values for each category. For independence tests, it shows the full contingency table with observed and expected values for each combination of row and column categories.

##### Show Method

```go
func (r *ChiSquareTestResult) Show()
```

**Description:** Displays the chi-square test results including the test statistic, p-value, degrees of freedom, and the contingency table.

**Parameters:**

- None.

**Returns:**

- None.

**Example**:

```go
// Goodness of fit test with categorical data
categoricalData := insyra.NewDataList("A", "B", "A", "C", "A", "B")
p := []float64{0.5, 0.3, 0.2} // Expected probabilities for A, B, C
result, err := stats.ChiSquareGoodnessOfFit(categoricalData, p, true)
if err != nil {
    log.Fatal(err)
}
result.Show() // Display complete test results

// Independence test
result, err = stats.ChiSquareIndependenceTest(rowData, colData)
if err != nil {
    log.Fatal(err)
}
result.Show() // Display complete test results with contingency table
```

---

## Distribution Analysis

### Skewness

```go
func Skewness(sample any, method ...SkewnessMethod) (float64, error)
```

**Description:** Calculate skewness (asymmetry) of a distribution.

**Parameters:**

- `sample`: Data (any type convertible to float64 slice)
- `method`: Optional skewness calculation method (default: SkewnessG1)

**Returns:**

- `float64`: Return value.

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
func Kurtosis(data any, method ...KurtosisMethod) (float64, error)
```

**Description:** Calculate kurtosis (tail heaviness) of a distribution.

**Parameters:**

- `data`: Data (any type convertible to float64 slice)
- `method`: Optional kurtosis calculation method (default: KurtosisG2)

**Returns:**

- `float64`: Return value.

#### Kurtosis Method

```go
type KurtosisMethod int
const (
    KurtosisG2           KurtosisMethod = iota + 1 // Type 1: g2 (default)
    KurtosisAdjusted                               // Type 2: adjusted Fisher kurtosis
    KurtosisBiasAdjusted                           // Type 3: bias-adjusted
)
```

**Returns**: Kurtosis value (float64). NaN if data is invalid.

### Calculate Moment

```go
func CalculateMoment(dl insyra.IDataList, n int, central bool) (float64, error)
```

**Description:** Calculate n-th moment of a dataset.

**Parameters:**

- `dl`: Input data list
- `n`: Moment order (positive integer)
- `central`: true for central moments, false for raw moments

**Returns:**

- `float64`: Return value.

**Example**:

```go
// Calculate skewness
skew, err := stats.Skewness(data, stats.SkewnessG1)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Skewness: %.4f\n", skew)

// Calculate kurtosis
kurt, err := stats.Kurtosis(data, stats.KurtosisG2)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Kurtosis: %.4f\n", kurt)

// Calculate 3rd central moment
moment3, err := stats.CalculateMoment(dataList, 3, true)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("3rd central moment: %.4f\n", moment3)
```

---

## Analysis of Variance (ANOVA)

### One Way ANOVA

```go
func OneWayANOVA(groups ...insyra.IDataList) (*OneWayANOVAResult, error)
```

**Description:** Compare means across multiple independent groups.

**Parameters:**

- `groups`: Variable number of data groups (minimum 2 groups)

**Returns:**

- `*OneWayANOVAResult`: Return value.

### Two Way ANOVA

```go
func TwoWayANOVA(factorALevels, factorBLevels int, cells ...insyra.IDataList) (*TwoWayANOVAResult, error)
```

**Description:** Analyze effects of two factors and their interaction.

**Parameters:**

- `factorALevels, factorBLevels`: Number of levels for each factor
- `cells`: Data for each factor combination

**Returns:**

- `*TwoWayANOVAResult`: Return value.

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
result, err := stats.OneWayANOVA(group1, group2, group3)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("F=%.4f, p=%.4f\n", result.Factor.F, result.Factor.P)

// Two-way ANOVA
cells := []insyra.IDataList{
    insyra.NewDataList(1, 2, 3),  // A1B1
    insyra.NewDataList(4, 5, 6),  // A1B2
    insyra.NewDataList(7, 8, 9),  // A2B1
    insyra.NewDataList(10, 11, 12), // A2B2
}
result, err = stats.TwoWayANOVA(2, 2, cells...)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Factor A F=%.4f, p=%.4f\n", result.FactorA.F, result.FactorA.P)
```

---

## F-Tests

### F-Test For Variance Equality

```go
func FTestForVarianceEquality(data1, data2 insyra.IDataList) (*FTestResult, error)
```

**Description:** Test equality of variances between two groups.

**Parameters:**

- `data1`: Input value for `data1`. Type: `insyra.IDataList`.
- `data2`: Input value for `data2`. Type: `insyra.IDataList`.

**Returns:**

- `*FTestResult`: Return value.

### Levene Test

```go
func LeveneTest(groups []insyra.IDataList) (*FTestResult, error)
```

**Description:** Test equality of variances across multiple groups (robust).

**Parameters:**

- `groups`: Input value for `groups`. Type: `[]insyra.IDataList`.

**Returns:**

- `*FTestResult`: Return value.

### Bartlett Test

```go
func BartlettTest(groups []insyra.IDataList) (*FTestResult, error)
```

**Description:** Test equality of variances across multiple groups (assumes normality).

**Parameters:**

- `groups`: Input value for `groups`. Type: `[]insyra.IDataList`.

**Returns:**

- `*FTestResult`: Return value.

### F-Test For Regression

```go
func FTestForRegression(ssr, sse float64, df1, df2 int) (*FTestResult, error)
```

**Description:** Test overall significance of regression model.

**Parameters:**

- `ssr`: Regression sum of squares
- `sse`: Error sum of squares
- `df1, df2`: Degrees of freedom

**Returns:**

- `*FTestResult`: Return value.

### F-Test For Nested Models

```go
func FTestForNestedModels(rssReduced, rssFull float64, dfReduced, dfFull int) (*FTestResult, error)
```

**Description:** Compare nested regression models.

**Parameters:**

- `rssReduced, rssFull`: Residual sum of squares for each model
- `dfReduced, dfFull`: Degrees of freedom for each model

**Returns:**

- `*FTestResult`: Return value.

#### F-Test Result

```go
type FTestResult struct {
    testResultBase
    DF2 float64 // Second degrees of freedom
}
```

**Example**:

```go
// Test variance equality
result, err := stats.FTestForVarianceEquality(group1, group2)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("F=%.4f, p=%.4f\n", result.Statistic, result.PValue)

// Levene's test
groups := []insyra.IDataList{group1, group2, group3}
result, err = stats.LeveneTest(groups)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Levene F=%.4f, p=%.4f\n", result.Statistic, result.PValue)

// Regression F-test
result, err = stats.FTestForRegression(1200.0, 800.0, 3, 20)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Regression F=%.4f, p=%.4f\n", result.Statistic, result.PValue)
```

---

## Principal Component Analysis (PCA)

### PCA

```go
func PCA(dataTable insyra.IDataTable, nComponents ...int) (*PCAResult, error)
```

**Description:** Perform principal component analysis to reduce dimensionality of data.

**Parameters:**

- `dataTable`: Input data table with observations in rows and variables in columns
- `nComponents`: Optional number of components to extract (default: all components)

**Returns:**

- `*PCAResult`: Return value.

#### PCA Result

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
result, err := stats.PCA(dataTable, 2)
if err != nil {
    log.Fatal(err)
}

// Access components and explained variance
components := result.Components
fmt.Printf("Explained variance: %.2f%%\n", result.ExplainedVariance[0])
```

## Clustering Analysis

## K-Nearest Neighbors (KNN)

### KNN Classification

```go
func KNNClassify(trainData insyra.IDataTable, trainLabels insyra.IDataList, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNClassificationResult, error)
```

**Description:** Predict categorical labels for `testData` by majority vote among the `k` nearest rows in `trainData`.

#### KNN Options

```go
type KNNWeighting string

const (
    KNNUniformWeighting  KNNWeighting = "uniform"
    KNNDistanceWeighting KNNWeighting = "distance"
)

type KNNAlgorithm string

const (
    KNNAuto       KNNAlgorithm = "auto"
    KNNBruteForce KNNAlgorithm = "brute"
    KNNKDTree     KNNAlgorithm = "kd_tree"
    KNNBallTree   KNNAlgorithm = "ball_tree"
)

type KNNOptions struct {
    Weighting KNNWeighting
    Algorithm KNNAlgorithm
    LeafSize  int
}
```

#### KNN Classification Result

```go
type KNNClassificationResult struct {
    Predictions   insyra.IDataList
    Classes       insyra.IDataList
    Probabilities insyra.IDataTable
}
```

**Notes:**

- v1 uses Euclidean distance
- default weighting is `uniform`
- default algorithm is `auto`
- `distance` weighting uses inverse distance
- exact-distance matches dominate distance-weighted predictions
- `auto` chooses exact brute-force / KD-tree / ball-tree search based on data shape

**Example**:

```go
trainLabels := insyra.NewDataList("red", "red", "red", "blue", "blue", "blue")
result, err := stats.KNNClassify(trainTable, trainLabels, testTable, 3, stats.KNNOptions{
    Weighting: stats.KNNDistanceWeighting,
    Algorithm: stats.KNNKDTree,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Predictions.Get(0))
result.Probabilities.Show()
```

### KNN Regression

```go
func KNNRegress(trainData insyra.IDataTable, trainTargets insyra.IDataList, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNRegressionResult, error)
```

**Description:** Predict numeric targets for `testData` by averaging the `k` nearest training targets.

#### KNN Regression Result

```go
type KNNRegressionResult struct {
    Predictions []float64
}
```

**Example**:

```go
targets := insyra.NewDataList(1.0, 1.5, 9.0, 9.5)
result, err := stats.KNNRegress(trainTable, targets, testTable, 2)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Predictions)
```

### K-Nearest Neighbor Search

```go
func KNearestNeighbors(trainData insyra.IDataTable, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNNeighborsResult, error)
```

**Description:** Return the `k` nearest training-row indices and distances for each row in `testData`.

#### KNN Neighbors Result

```go
type KNNNeighborsResult struct {
    Indices   [][]int
    Distances [][]float64
}
```

**Notes:**

- returned indices are 1-based row indices
- `Distances[i][j]` matches `Indices[i][j]`
- all backends return exact nearest neighbors, not approximations

**Example**:

```go
neighbors, err := stats.KNearestNeighbors(trainTable, testTable, 3, stats.KNNOptions{
    Algorithm: stats.KNNBallTree,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(neighbors.Indices)
fmt.Println(neighbors.Distances)
```

### KMeans

```go
func KMeans(dataTable insyra.IDataTable, centers int, opts ...KMeansOptions) (*KMeansResult, error)
```

**Description:** Partition observations into `centers` clusters and return cluster labels, centers, and sum-of-squares summaries.

**Parameters:**

- `dataTable`: Numeric input data with observations in rows
- `centers`: Number of clusters (`1 <= centers <= nrow`)
- `opts`: Optional `KMeansOptions` (at most one value)

#### KMeans Options

```go
type KMeansOptions struct {
    NStart  int
    IterMax int
    Seed    *int64
}
```

#### KMeans Result

```go
type KMeansResult struct {
    Cluster     []int
    Centers     insyra.IDataTable
    TotSS       float64
    WithinSS    []float64
    TotWithinSS float64
    BetweenSS   float64
    Size        []int
    Iter        int
    IFault      int
}
```

**Example**:

```go
seed := int64(7)
result, err := stats.KMeans(dataTable, 3, stats.KMeansOptions{
    NStart:  5,
    IterMax: 25,
    Seed:    &seed,
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Cluster)
fmt.Println(result.Size)
result.Centers.Show()
```

### Hierarchical Agglomerative Clustering

```go
func HierarchicalAgglomerative(dataTable insyra.IDataTable, method AgglomerativeMethod) (*HierarchicalResult, error)
func CutTreeByK(tree *HierarchicalResult, k int) ([]int, error)
func CutTreeByHeight(tree *HierarchicalResult, h float64) ([]int, error)
```

#### Agglomerative Method

```go
type AgglomerativeMethod string
const (
    AggloComplete AgglomerativeMethod = "complete"
    AggloSingle   AgglomerativeMethod = "single"
    AggloAverage  AgglomerativeMethod = "average"
    AggloWardD    AgglomerativeMethod = "ward.D"
    AggloWardD2   AgglomerativeMethod = "ward.D2"
    AggloMcQuitty AgglomerativeMethod = "mcquitty"
    AggloMedian   AgglomerativeMethod = "median"
    AggloCentroid AgglomerativeMethod = "centroid"
)
```

#### Hierarchical Result

```go
type HierarchicalResult struct {
    Merge      [][2]int
    Height     []float64
    Order      []int
    Labels     []string
    Method     AgglomerativeMethod
    DistMethod string
}
```

**Example**:

```go
tree, err := stats.HierarchicalAgglomerative(dataTable, stats.AggloComplete)
if err != nil {
    log.Fatal(err)
}
labels, err := stats.CutTreeByK(tree, 3)
if err != nil {
    log.Fatal(err)
}
fmt.Println(labels)
```

### DBSCAN

```go
func DBSCAN(dataTable insyra.IDataTable, eps float64, minPts int, opts ...DBSCANOptions) (*DBSCANResult, error)
```

#### DBSCAN Options

```go
type DBSCANOptions struct {
    BorderPoints *bool
}
```

#### DBSCAN Result

```go
type DBSCANResult struct {
    Cluster []int
    IsSeed  []bool
}
```

**Example**:

```go
result, err := stats.DBSCAN(dataTable, 0.35, 4)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Cluster)
fmt.Println(result.IsSeed)
```

### Silhouette

```go
func Silhouette(dataTable insyra.IDataTable, labels insyra.IDataList) (*SilhouetteResult, error)
```

#### Silhouette Result

```go
type SilhouettePoint struct {
    Cluster  int
    Neighbor int
    SilWidth float64
}

type SilhouetteResult struct {
    Points            []SilhouettePoint
    AverageSilhouette float64
}
```

**Example**:

```go
labels := insyra.NewDataList(1, 1, 2, 2, 3, 3)
result, err := stats.Silhouette(dataTable, labels)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Average silhouette: %.4f\n", result.AverageSilhouette)
```

## Regression Analysis

### Linear Regression

```go
func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LinearRegressionResult, error)
```

**Description:** Performs ordinary least-squares linear regression. Supports both simple regression (one predictor) and multiple regression (multiple predictors).

**Parameters:**

- `dlY`: Dependent variable.
- `dlXs`: One or more independent variables. All inputs must have the same length.

**Returns:**

- `*LinearRegressionResult`: Model coefficients, inference statistics, residuals, and fit metrics.

#### Linear Regression Result

```go
type LinearRegressionResult struct {
    // Simple-regression convenience fields (when len(dlXs) == 1)
    Slope                  float64
    Intercept              float64
    StandardError          float64
    StandardErrorIntercept float64
    TValue                 float64
    TValueIntercept        float64
    PValue                 float64
    PValueIntercept        float64

    ConfidenceIntervalIntercept [2]float64
    ConfidenceIntervalSlope     [2]float64

    // General fields (available for both simple and multiple regression)
    Coefficients        []float64
    StandardErrors      []float64
    TValues             []float64
    PValues             []float64
    ConfidenceIntervals [][2]float64

    Residuals        []float64
    RSquared         float64
    AdjustedRSquared float64
}
```

**Example (simple regression):**

```go
y := insyra.NewDataList([]float64{1, 2, 3, 4, 5})
x := insyra.NewDataList([]float64{2, 4, 5, 4, 5})

result, err := stats.LinearRegression(y, x)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("intercept=%.4f (p=%.4f)\n", result.Intercept, result.PValueIntercept)
fmt.Printf("slope=%.4f (p=%.4f)\n", result.Slope, result.PValue)
fmt.Printf("R-squared=%.4f\n", result.RSquared)
fmt.Printf("95%% CI slope=[%.4f, %.4f]\n", result.ConfidenceIntervalSlope[0], result.ConfidenceIntervalSlope[1])
```

**Example (multiple regression):**

```go
y := insyra.NewDataList([]float64{15, 25, 30, 35, 40, 50})
x1 := insyra.NewDataList([]float64{2, 3, 4, 5, 6, 7})
x2 := insyra.NewDataList([]float64{1, 2, 2, 3, 3, 4})

result, err := stats.LinearRegression(y, x1, x2)
if err != nil {
    log.Fatal(err)
}

for i := range result.Coefficients {
    fmt.Printf("beta[%d]=%.4f, p=%.4f, CI=[%.4f, %.4f]\n",
        i,
        result.Coefficients[i],
        result.PValues[i],
        result.ConfidenceIntervals[i][0],
        result.ConfidenceIntervals[i][1],
    )
}
```

### Polynomial Regression

```go
func PolynomialRegression(dlY insyra.IDataList, dlX insyra.IDataList, degree int) (*PolynomialRegressionResult, error)
```

**Description:** Fits a polynomial model:

```text
y = a0 + a1*x + a2*x^2 + ... + ak*x^k
```

**Parameters:**

- `dlY`: Dependent variable.
- `dlX`: Independent variable.
- `degree`: Polynomial degree (`>= 1`).

**Returns:**

- `*PolynomialRegressionResult`: Coefficients, inference statistics, residuals, and fit metrics.

#### Polynomial Regression Result

```go
type PolynomialRegressionResult struct {
    Coefficients        []float64
    Degree              int
    Residuals           []float64
    RSquared            float64
    AdjustedRSquared    float64
    StandardErrors      []float64
    TValues             []float64
    PValues             []float64
    ConfidenceIntervals [][2]float64
}
```

**Example:**

```go
result, err := stats.PolynomialRegression(yData, xData, 3)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("R-squared=%.4f\n", result.RSquared)
```

### Exponential Regression

```go
func ExponentialRegression(dlY insyra.IDataList, dlX insyra.IDataList) (*ExponentialRegressionResult, error)
```

**Description:** Fits an exponential model:

```text
y = a * exp(b*x)
```

**Parameters:**

- `dlY`: Dependent variable (`y > 0` required).
- `dlX`: Independent variable.

**Returns:**

- `*ExponentialRegressionResult`: Coefficients, inference statistics, confidence intervals, residuals, and fit metrics.

#### Exponential Regression Result

```go
type ExponentialRegressionResult struct {
    Intercept              float64
    Slope                  float64
    Residuals              []float64
    RSquared               float64
    AdjustedRSquared       float64
    StandardErrorIntercept float64
    StandardErrorSlope     float64
    TValueIntercept        float64
    TValueSlope            float64
    PValueIntercept        float64
    PValueSlope            float64

    ConfidenceIntervalIntercept [2]float64
    ConfidenceIntervalSlope     [2]float64
}
```

### Logarithmic Regression

```go
func LogarithmicRegression(dlY insyra.IDataList, dlX insyra.IDataList) (*LogarithmicRegressionResult, error)
```

**Description:** Fits a logarithmic model:

```text
y = a + b*ln(x)
```

**Parameters:**

- `dlY`: Dependent variable.
- `dlX`: Independent variable (`x > 0` required).

**Returns:**

- `*LogarithmicRegressionResult`: Coefficients, inference statistics, confidence intervals, residuals, and fit metrics.

#### Logarithmic Regression Result

```go
type LogarithmicRegressionResult struct {
    Intercept              float64
    Slope                  float64
    Residuals              []float64
    RSquared               float64
    AdjustedRSquared       float64
    StandardErrorIntercept float64
    StandardErrorSlope     float64
    TValueIntercept        float64
    TValueSlope            float64
    PValueIntercept        float64
    PValueSlope            float64

    ConfidenceIntervalIntercept [2]float64
    ConfidenceIntervalSlope     [2]float64
}
```

---

## Matrix Operations

### Diag

```go
func Diag(x any, dims ...int) (any, error)
```

**Description:** Create diagonal matrices or extract diagonal elements from matrices, mimicking R's `diag()` function.

**Parameters:**

- `x`: Input value of various types:
  - `*mat.Dense`: Extract diagonal elements as `[]float64`
  - `[]float64`: Create diagonal matrix from slice
  - `int` or `float64`: Create identity matrix of specified size
  - `nil`: Create identity matrix (default 1x1)
- `dims`: Optional dimensions (0, 1, or 2 values):
  - No dims: Use default sizing based on input
  - 1 dim: Set nrow = ncol = dim[0]
  - 2 dims: Set nrow = dim[0], ncol = dim[1]

**Returns:**

- When extracting: `[]float64` containing diagonal elements
- When creating: `*mat.Dense` diagonal or identity matrix

**Examples**:

```go
// Extract diagonal from matrix
matrix := mat.NewDense(3, 3, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9})
diagonal, err := Diag(matrix) // Returns []float64{1, 5, 9}
if err != nil {
    log.Fatal(err)
}

// Create diagonal matrix from slice
values := []float64{1, 2, 3}
diagMatrix, err := Diag(values) // Returns 3x3 diagonal matrix
if err != nil {
    log.Fatal(err)
}

// Create identity matrix
identity, err := Diag(3) // Returns 3x3 identity matrix
if err != nil {
    log.Fatal(err)
}

// Create rectangular identity matrix
rectIdentity, err := Diag(nil, 2, 3) // Returns 2x3 matrix with diagonal 1s
if err != nil {
    log.Fatal(err)
}
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
CI = coefficient +/- t_(alpha/2, df) * standard_error
```

Where:

- `t_(alpha/2, df)` is the critical value from the t-distribution
- `df` is the degrees of freedom (sample_size - number_of_parameters)
- `alpha = 0.05` for 95% confidence intervals

### Data Input Types

Functions accept various data input types:

- `insyra.IDataList`: Primary interface for data lists
- `any`: Raw data that can be converted to float64 slices
- `[]float64`: Direct float64 slices where applicable

### Error Handling

Functions return `error` values when:

- Input data is empty or invalid
- Sample sizes are too small for the test
- Mathematical requirements are not met
- Invalid parameter combinations are provided

Call sites should always check `err` and handle it explicitly.

---

## Behavior Differences From R

The numerical output of insyra's `stats` package agrees with R's standard
functions (`t.test`, `cor.test`, `aov`, `prcomp`, `kmeans`, `dbscan`,
`lm`, `chisq.test`, `var.test`, `bartlett.test`, median-centered
`car::leveneTest`) to within ~1e-12 on every well-defined numerical
field, with the single semantic exception below. Discrete outputs (DF,
cluster IDs, hclust merge structure) match exactly.

### Behavior on degenerate / constant inputs

R's `t.test`, `cor.test`, and similar functions abort with an error
when the input is essentially constant (zero variance, singleton
cluster, etc.). insyra returns **sentinel values** instead so callers
can handle the case without recovering from a panic:

| Function | Degenerate input | insyra returns | R behavior |
|---|---|---|---|
| `SingleSampleTTest` | constant data, `mean == μ` | `t = NaN, p = NaN, d = 0, CI = [μ, μ]` | error |
| `SingleSampleTTest` | constant data, `mean ≠ μ` | `t = ±Inf, p = 0, d = ±Inf, CI = [mean, mean]` | error |
| `Correlation` (any method) | one variable has zero variance | error (`"cannot calculate correlation due to zero variance"`) | `NA` with warning |
| `Silhouette` | only one cluster | error | not defined |

If you are migrating a script from R: wrap insyra calls' results in a
`!math.IsNaN(t)` guard rather than expecting a panic.

## Inference Extensions Beyond R

Several insyra return values have no direct R counterpart — R's standard
functions don't return them, so there is no R reference to "agree" or
"disagree" with. We document insyra's formula choices here so you can
compare against textbooks or other packages (SPSS, scipy, Pingouin).

### Cohen's d (effect size on t-tests / z-tests)

R's `t.test` / `BSDA::z.test` don't return effect size at all. insyra
populates `EffectSizes[0]` ("`cohen_d`") for every t-test and z-test
variant using the formulas below — chosen to match the textbook
convention for each design.

| Test variant | insyra formula | Note |
|---|---|---|
| `SingleSampleTTest` | `(mean − μ) / sd` | Standard one-sample d, sign preserved. |
| `TwoSampleTTest` (equal var) | `(m1 − m2) / sqrt(pooledVar)`, `pooledVar = ((n1−1)v1 + (n2−1)v2) / (n1+n2−2)` | Classical Cohen's d using sample-pooled SD. |
| `TwoSampleTTest` (Welch) | `(m1 − m2) / sqrt((v1 + v2) / 2)` | Cohen's d_av — average-variance variant; pooled SD assumes equal variance. |
| `PairedTTest` | `meanDiff / sd(diff)` | Cohen's d_z, sign preserved. |
| `SingleSampleZTest` | `\|mean − μ\| / σ` | Uses known population σ. Sign carried by the z-statistic. |
| `TwoSampleZTest` | `\|m1 − m2\| / sqrt((n1·σ1² + n2·σ2²) / (n1+n2))` | Sample-size-weighted pooled population σ. Differs from the textbook `sqrt((σ1² + σ2²) / 2)` by weighting each population variance by its observed sample size. |

### ANOVA partial η²

`OneWayANOVA` / `TwoWayANOVA` populate `EtaSquared` per factor as

```text
η²_partial = SS_effect / (SS_effect + SS_within)
```

R's `aov()` does not give η² directly; SPSS reports partial η² by
default. For one-way ANOVA partial η² equals classical η² (`SS_effect /
SS_total`); for two-way ANOVA they differ.

`RepeatedMeasuresANOVA` is the one exception — its `Factor.EtaSquared`
is **classical** η² (`SS_factor / SS_total`), since partial η² for
within-subjects designs has multiple competing definitions.

### Spearman correlation confidence interval

R's `cor.test(method="spearman")` does not return a CI by default.
insyra applies the same **Fisher z-transform** CI used for Pearson:

```text
z = atanh(r),  se = 1 / sqrt(n − 3)
[lower, upper] = tanh([z − z_crit·se, z + z_crit·se])
```

with boundary cases `r ≥ 1 → [1, 1]`, `r ≤ −1 → [−1, −1]`, and
`n ≤ 3 → [NaN, NaN]`. For Pearson this is exactly what R's `cor.test`
returns; for Spearman it's an insyra-only addition (the asymptotic
distribution of Fisher-z(ρ) is normal under H₀ and approximately so
under any null, so the same CI formula is defensible).

## Algorithm Notes

These sections describe non-obvious implementation choices that match
R but may surprise readers expecting other packages' defaults.

### Spearman p-value (matches R `cor.test`)

`Correlation(..., SpearmanCorrelation)` is a port of R's
`cor.test(method="spearman")` p-value path:

| n | Algorithm |
|---|---|
| 2 ≤ n ≤ 9 | Exact enumeration of all n! rank permutations |
| 10 ≤ n ≤ 1290 | AS-89 Edgeworth-series approximation (Best & Roberts 1975) |
| n > 1290, or any n with ties | Fisher r-to-t — `t = ρ · √(n−2) / √(1−ρ²)` |

The exact and AS-89 paths follow R's `prho.c` byte-for-byte. SciPy's
`stats.spearmanr` uses Fisher r-to-t universally and disagrees with R
(and insyra) for small n without ties — its p-value is much smaller
than the discrete exact distribution allows.

### Kendall correlation (matches R `cor.test`)

`Correlation(..., KendallCorrelation)` returns Kendall's τ-b with
tie correction:

```text
τ_b = (concordant − discordant) / sqrt((n0 − n1) · (n0 − n2))
       n0 = n(n − 1) / 2
       n1 = Σ tx(tx − 1) / 2  (tie groups in X)
       n2 = Σ ty(ty − 1) / 2  (tie groups in Y)
```

The p-value uses exact two-sided permutation for n ≤ 7 and the full
Kendall (1948) tie-corrected asymptotic for n > 7:

```text
z = S / sqrt(var(S))                          S = concordant − discordant
var(S) = (n(n − 1)(2n + 5) − T1 − T2) / 18    ← first-order
       + (T1b · T2b) / (9 n(n − 1)(n − 2))    ← second-order (cross)
       + (T1c · T2c) / (2 n(n − 1))           ← second-order (pair)
  Ti  = Σ t(t − 1)(2t + 5)
  Tib = Σ t(t − 1)(t − 2)
  Tic = Σ t(t − 1)
```

τ-b and the p-value match `cor.test(method="kendall")` to machine
precision. SciPy's `stats.kendalltau` historically dropped the
second-order variance corrections and so disagrees with both R and
insyra by a small amount on tied data.

We do **not** wrap `gonum/stat.Kendall` because gonum returns τ-a
(no tie correction in the denominator). With ties present, |τ-a| can
fall short of 1 even for a perfectly monotonic relationship and
disagrees with virtually every other stats package. Self-implementing
τ-b is ~50 lines in `stats/correlation.go:kendallTauBStats`.


