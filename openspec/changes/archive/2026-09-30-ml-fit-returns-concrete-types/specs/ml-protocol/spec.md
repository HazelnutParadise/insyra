## ADDED Requirements

### Requirement: A fit function returns the model's own type

Every exported `Fit*` function in `ml` SHALL return a pointer to the model type it builds, never an interface: `FitLinearRegression` `*LinearModel`, `FitPolynomialRegression` `*PolynomialModel`, `FitWeightedLinearRegression` `*WeightedLinearModel`, `FitRidgeRegression` `*RidgeModel`, `FitLassoRegression` `*LassoModel`, `FitExponentialRegression` `*ExponentialModel`, `FitLogarithmicRegression` `*LogarithmicModel`, `FitLogisticRegression` `*LogisticModel`, `FitPoissonRegression` `*PoissonModel`, `FitGLM` `*GLMModel`, `FitKMeans` `*KMeansModel`, `FitPCA` `*PCATransformer`, `FitKNNClassifier` `*KNNClassifier`, `FitKNNRegressor` `*KNNRegressor`, and the tree and ensemble functions their own types as before. Each type SHALL implement every protocol interface the function returned before the change. `Estimator.Fit`, `Estimator.FitWeighted` and `Step.Fit` SHALL keep returning `Model` and `Transformer`.

#### Scenario: The wrapped result is read without a type assertion
- **WHEN** 呼叫 `ml.FitLinearRegression(x, y)` 並直接讀 `model.Result.Coefficients`
- **THEN** 程式可以編譯，係數與直接呼叫 `stats.LinearRegression` 的結果相同

#### Scenario: A returned type still satisfies its interface
- **WHEN** 把 `FitLogisticRegression` 或 `FitKNNClassifier` 的結果指派給 `ml.ProbaModel`，把 `FitKMeans` 的結果指派給 `ml.Clusterer`，把 `FitPCA` 的結果指派給 `ml.Transformer`
- **THEN** 都可以編譯，行為與改動前相同

#### Scenario: A fit function inside an estimator
- **WHEN** 把 `ml.FitLinearRegression` 放進 `ml.Estimator` 的 `Fit`
- **THEN** 要寫成回傳 `(ml.Model, error)` 的 closure，交叉驗證結果與改動前相同

### Requirement: A fitted model is safe to call when it is nil

Every exported method of every type an `ml` `Fit*` function returns SHALL be safe to call on a nil receiver: `Features`, `FeatureImportances` and `LeafValues` SHALL return nil, `Clusters` SHALL return 0, `Classes` SHALL return an empty list whose `Err()` is set, and `Predict`, `PredictProba`, `Transform` and `ExportONNX` SHALL return an error. `ml.ExportONNX` SHALL return an error, and write nothing, when its model argument is nil or a nil pointer.

#### Scenario: A failed fit reached through an interface
- **WHEN** 一個宣告回傳 `(ml.Model, error)` 的 closure 回傳 `ml.FitLinearRegression(nil, y)` 的結果，呼叫者忽略錯誤，對拿到的 model 呼叫 `Features` 與 `Predict`
- **THEN** 不 panic，`Features` 回傳 nil，`Predict` 回傳錯誤

#### Scenario: Every method of every returned type on nil
- **WHEN** 對二十種 `Fit*` 回傳型別的 nil 指標呼叫它們的每個匯出方法
- **THEN** 沒有任何 panic，每個方法都回傳錯誤或空值

#### Scenario: Exporting a nil model
- **WHEN** 呼叫 `ml.ExportONNX(w, (*ml.LinearModel)(nil))`
- **THEN** 回傳錯誤，沒有寫入 `w`，不 panic
