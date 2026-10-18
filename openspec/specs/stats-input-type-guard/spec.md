# stats-input-type-guard Specification

## Purpose
`stats` 對介面參數的處理契約：經 `asDataList` 或 `numericSlice` 讀取輸入的函式接受任何 `IDataList` 實作，nil 介面值回錯誤而非 panic；帶型別的 nil 只有假設檢定有保證（見 `stats-test-input`），其他函式尚未納入。
## Requirements
### Requirement: Interface parameters are converted, not asserted

經 `asDataList` 或 `numericSlice` 讀取 `insyra.IDataList` 參數的 `stats` 函式，即 `Correlation`、`Covariance`、`FTestForVarianceEquality`、`TwoSampleTTest`、`PairedTTest`、`TwoSampleZTest`、`MannWhitneyU`、`PairedWilcoxon`、`LinearRegression`、`WeightedLinearRegression`、`RidgeRegression`、`LassoRegression`、`ExponentialRegression`、`LogarithmicRegression`、`PolynomialRegression`、`GLM`、`LogisticRegressionWithOptions`、`PoissonRegressionWithOptions`，SHALL 接受任何實作，參數為 nil 介面值時 SHALL 回傳錯誤而 SHALL NOT panic。

#### Scenario: Nil input
- **WHEN** `Correlation(nil, NewDataList(1.0, 2.0), PearsonCorrelation)`
- **THEN** 回傳錯誤，不 panic

### Requirement: PCA and KNNRegress refuse non-finite values
The system SHALL return an error from `PCA` when a cell of the table is NaN or infinite, and from `KNNRegress` when a target is NaN or infinite, instead of returning a result built from it.

#### Scenario: PCA with a NaN cell
- **WHEN** `PCA` receives a table with a NaN in one cell
- **THEN** it returns an error and no result

#### Scenario: KNNRegress with an infinite target
- **WHEN** `KNNRegress` receives a target list holding +Inf
- **THEN** it returns an error and no prediction

