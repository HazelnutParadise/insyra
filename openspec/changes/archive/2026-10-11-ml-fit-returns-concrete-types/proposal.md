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

- The `Fit*` functions keep returning the interfaces they return today. The change that makes them return concrete types is left on 0.4 (BREAKING), with `Estimator.Fit` closures.
- `Predict` on a nil pointer of each regression model (`LinearModel`, `WeightedLinearModel`, `RidgeModel`, `LassoModel`, `PolynomialModel`, `ExponentialModel`, `LogarithmicModel`, `LogisticModel`, `PoissonModel`, `GLMModel`) returns an error instead of panicking.
- `ml.ExportONNX` refuses a nil model, or a nil pointer inside its argument, with an error and writes nothing. Every model's `ExportONNX` method goes through it.
- Left on 0.4: nil-safe `Features` methods on the pointer types. On this line `Features` stays a value method promoted from `modelBase`, so a `Features` call on a nil pointer still panics; a nil-checking method on each pointer type would hide the value method from interfaces that name `Features`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ml-protocol`: adds "A nil fitted regression model refuses to predict", "ExportONNX refuses a nil model without writing" and "Features stays a value method".

## Impact

- `ml/models.go`, `ml/onnx_export.go`, and the `ml` tests.
- `Docs/ml.md`, `CHANGELOG.md` and `CHANGELOG_TW.md`.
