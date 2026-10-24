# ml-protocol Specification

## Purpose
The one interface every `ml` model and preprocessing step satisfies, so pipelines, cross-validation and export can take any of them: wrapped `stats` models keep their results, capabilities beyond prediction are discoverable, a third party can check its own model, and a nil fitted model refuses to predict or export.


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

### Requirement: A nil fitted regression model refuses to predict

`Predict` on a nil pointer of a regression model type SHALL return an error instead of panicking. The types are `LinearModel`, `WeightedLinearModel`, `RidgeModel`, `LassoModel`, `PolynomialModel`, `ExponentialModel`, `LogarithmicModel`, `LogisticModel`, `PoissonModel` and `GLMModel`.

#### Scenario: Predicting with a nil model
- **WHEN** 對 `(*ml.LinearModel)(nil)` 呼叫 `Predict`
- **THEN** 回傳錯誤，不 panic

### Requirement: ExportONNX refuses a nil model without writing

`ml.ExportONNX` SHALL return an error and write nothing when its model argument is nil or a nil pointer, and every model's `ExportONNX` method SHALL do the same because it calls `ml.ExportONNX`.

#### Scenario: Exporting a nil model
- **WHEN** 呼叫 `ml.ExportONNX(w, (*ml.LinearModel)(nil))`
- **THEN** 回傳錯誤，沒有寫入 `w`，不 panic

### Requirement: Features stays a value method

`Features` SHALL remain a method promoted from the embedded model base, so that a model value, not only a pointer, satisfies an interface naming `Features`.

#### Scenario: A model value satisfies an interface naming Features
- **WHEN** 把 `ml.LinearModel{}` 或 `ml.DecisionTreeClassifier{}` 指派給宣告 `Features() []string` 的介面
- **THEN** 可以編譯
