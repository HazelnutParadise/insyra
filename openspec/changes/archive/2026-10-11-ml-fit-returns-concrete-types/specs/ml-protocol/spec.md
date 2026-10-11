## ADDED Requirements

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
