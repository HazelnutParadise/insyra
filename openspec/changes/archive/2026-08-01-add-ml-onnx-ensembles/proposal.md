# Change: Export the new model families to ONNX

## Why
Seven model families landed since the exporter was written — ridge, lasso, WLS, both forests, both boosters — and none of them can leave the process. The flagship tabular models are exactly the ones a deployment pipeline wants as ONNX, and the operators they need already exist: the penalized and weighted linear fits are `LinearRegressor` with different coefficients, and `TreeEnsemble` was designed for many trees — the single-tree exporter has been writing a one-tree ensemble all along, with the tree id hardcoded to zero.

The verification story is what makes this change cheap to trust: the round trip against a real `onnxruntime` runs in CI now, so every new family is proved by execution, not by construction.

## What Changes
- Export `RidgeModel`, `LassoModel` and `WeightedLinearModel` through the existing `LinearRegressor` path — their coefficient layout already matches
- Teach the tree-ensemble builder multiple trees and a leaf scale: forests write every tree with its own id and scale leaf contributions by 1/T, so the runtime's sum is the forest's average
- Export `GradientBoostingRegressor` with the learning rate baked into the leaf weights and the mean as the ensemble base value — the runtime's sum is exactly the boosted prediction
- Export `GradientBoostingClassifier` as its raw log-odds ensemble with the runtime applying the logistic transform, the second class carrying the scores and the first its complement
- Extend the independent-runtime round trip with every new family, numeric outputs within single-precision tolerance and labels exact
- Refuse nothing new: models the exporter does not support keep the existing refusal

## Impact
- Affected specs: `ml-onnx`
- Affected code: `ml/onnx_export.go`, `ml/onnx_export_test.go`, docs, changelogs, `skills/insyra/`
- Additive; existing exports are unchanged
