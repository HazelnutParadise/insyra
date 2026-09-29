# Proposal: ml-fit-returns-concrete-types

## Why

ML-1 of the API review ([#263](https://github.com/HazelnutParadise/insyra/issues/263)): the fourteen `Fit*` functions that wrap `stats` return an interface (`Model`, `ProbaModel` or `Transformer`), while the six tree and ensemble `Fit*` functions return their own types. The wrapped models keep the `stats` result in an exported `Result` field, so a caller who wants the coefficients of the model they just fitted has to type-assert first:

```go
model, _ := ml.FitLinearRegression(x, y)
coefficients := model.(*ml.LinearModel).Result.Coefficients
```

The package does the same job two ways, and the interface return hides a field the package documents.

Returning the concrete type makes one thing newly reachable. A caller that assigns the result to an interface, for example a fit closure `return ml.FitLinearRegression(x, y)` declared to return `(ml.Model, error)`, gets a non-nil `Model` holding a nil pointer when the fit fails. Measured on 2026-09-30 against `origin/0.4` (7228ed2e): `Features`, `Predict` and `ExportONNX` on a nil `*LinearModel` panic, so does `ml.ExportONNX(w, (*ml.LinearModel)(nil))`, and `Features` on a nil `*DecisionTreeClassifier` or `*KMeansModel` panics too, although the tree types have returned concrete pointers all along. The library does not panic on caller input, so the types a `Fit*` function returns have to be safe to call on nil.

## What Changes

- **BREAKING**: every `Fit*` function returns its own type: `FitLinearRegression` `*LinearModel`, `FitPolynomialRegression` `*PolynomialModel`, `FitWeightedLinearRegression` `*WeightedLinearModel`, `FitRidgeRegression` `*RidgeModel`, `FitLassoRegression` `*LassoModel`, `FitExponentialRegression` `*ExponentialModel`, `FitLogarithmicRegression` `*LogarithmicModel`, `FitLogisticRegression` `*LogisticModel`, `FitPoissonRegression` `*PoissonModel`, `FitGLM` `*GLMModel`, `FitKMeans` `*KMeansModel`, `FitPCA` `*PCATransformer`, `FitKNNClassifier` `*KNNClassifier`, `FitKNNRegressor` `*KNNRegressor`. Each type still implements the interface the function used to return, which the package asserts at compile time. Code that assigns the result to a variable of the interface type keeps compiling; `Fit: ml.FitLinearRegression` in an `Estimator` no longer does, because a function value's result types must match exactly, and becomes a closure.
- `Estimator.Fit`, `Estimator.FitWeighted` and `Step.Fit` keep returning `Model` and `Transformer`: they are the protocol's seam, where one pipeline holds any model.
- Every method on every type a `Fit*` function returns, the six tree and ensemble types included, is safe on a nil receiver: `Features`, `FeatureImportances` and `LeafValues` return nil, `Clusters` returns 0, `Classes` returns the empty list with an `Err()` it already returns for an unfitted model, and `Predict`, `PredictProba`, `Transform` and `ExportONNX` return an error without writing anything.
- `ml.ExportONNX` refuses a nil pointer inside its argument with an error instead of panicking.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ml-protocol`: adds "A fit function returns the model's own type" and "A fitted model is safe to call when it is nil".

## Impact

- `ml/models.go`, `ml/decision_tree.go`, `ml/random_forest.go`, `ml/gradient_boosting.go`, `ml/onnx_export.go`, and the `ml` tests; `nn` tests that call `ml.Fit*` compile unchanged.
- `Docs/ml.md`, both changelogs, `api-review.md` (ML-1), `delivery-status.md`.
