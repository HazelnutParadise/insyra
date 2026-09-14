# stats-input-type-guard Specification

## Purpose
`stats` 對介面參數的處理契約：經 `asDataList` 或 `numericSlice` 讀取輸入的函式接受任何 `IDataList` 實作，nil 介面值回錯誤而非 panic；其他函式尚未納入。

## Requirements
### Requirement: Interface parameters are converted, not asserted

經 `asDataList` 或 `numericSlice` 讀取 `insyra.IDataList` 參數的 `stats` 函式，即 `Correlation`、`Covariance`、`FTestForVarianceEquality`、`TwoSampleTTest`、`PairedTTest`、`TwoSampleZTest`、`MannWhitneyU`、`PairedWilcoxon`、`LinearRegression`、`WeightedLinearRegression`、`RidgeRegression`、`LassoRegression`、`ExponentialRegression`、`LogarithmicRegression`、`PolynomialRegression`、`GLM`、`LogisticRegressionWithOptions`、`PoissonRegressionWithOptions`，SHALL 接受任何實作，參數為 nil 介面值時 SHALL 回傳錯誤而 SHALL NOT panic。

#### Scenario: Nil input
- **WHEN** `Correlation(nil, NewDataList(1.0, 2.0), PearsonCorrelation)`
- **THEN** 回傳錯誤，不 panic

