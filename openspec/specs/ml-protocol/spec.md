# ml-protocol Specification

## Purpose
TBD - created by archiving change add-ml-estimator-protocol. Update Purpose after archive.

## Requirements

### Requirement: One protocol over every model
The system SHALL expose every fitted model through one interface, whatever algorithm produced it.

#### Scenario: Any model scores new observations
- **WHEN** a caller holds a fitted model obtained through the package
- **THEN** it can score new observations through the same method regardless of which algorithm fitted it

#### Scenario: A model is asked which columns it was fit on
- **WHEN** a caller holds a fitted model
- **THEN** the columns it was fitted on are readable from it, in the order it expects them

#### Scenario: New observations carry different columns than the fit
- **WHEN** observations are scored whose columns do not match what the model was fitted on
- **THEN** the request is refused with an error naming the mismatch
- **AND** the columns are matched by name rather than by position

### Requirement: Existing preprocessing satisfies the protocol unchanged
The system SHALL accept the scalers and encoders the root package already provides as protocol members without adaptation.

#### Scenario: A scaler or encoder is used as a transformer
- **WHEN** a caller uses one of the root package's fitted scalers or encoders where a transformer is expected
- **THEN** it is accepted with no wrapping
- **AND** its behaviour is unchanged from calling it directly

### Requirement: Wrapped models return what the wrapped function returns
The system SHALL produce, for any model it wraps, the same numbers the underlying `stats` function produces.

#### Scenario: A wrapped model is compared against the function it wraps
- **WHEN** a model fitted through the protocol is compared against the same fit performed directly
- **THEN** the coefficients, predictions and reported statistics are identical, not merely close

#### Scenario: The underlying fit fails
- **WHEN** the wrapped function returns an error
- **THEN** the error is returned unchanged rather than being replaced or swallowed

### Requirement: Capabilities beyond prediction are discoverable
The system SHALL let a caller discover whether a fitted model supports a capability rather than assuming it.

#### Scenario: A model that reports class probabilities
- **WHEN** a caller needs class probabilities
- **THEN** it can determine whether a given model provides them before asking
- **AND** the column order of the probabilities matches the order in which the model reports its classes

#### Scenario: A model that does not support a capability
- **WHEN** a caller checks for a capability a model does not have
- **THEN** the check reports its absence rather than failing at call time

### Requirement: A third party can check its own model
The system SHALL provide a way to verify that a model outside the package obeys the protocol.

#### Scenario: An external model is checked
- **WHEN** a model implemented outside the package is put through the conformance check
- **THEN** every rule the protocol states is exercised against it
- **AND** a violation is reported naming the rule that was broken

### Requirement: Penalized regression is fitted through the protocol
The system SHALL expose ridge and lasso fitting through the same fitting-function shape as the other regression families, returning models that score new observations by feature name.

#### Scenario: A penalized model is fitted and scores new data

- **WHEN** a caller fits ridge or lasso through the package on a feature table
- **THEN** the returned model predicts through the same method as every other model
- **AND** it passes the protocol conformance checks

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
