# Change: The `insyra/ml` estimator protocol, over the models `stats` already has

## Why
`stats` implements most of a classical machine-learning toolkit — seven regression families, four clustering algorithms, PCA, factor analysis, KNN — and every one of them has a different shape. `LinearRegression` returns a `*LinearRegressionResult`, `KMeans` a `*KMeansResult`, `PCA` a `*PCAResult`. Nothing composes, because nothing shares a form.

scikit-learn's real contribution was never the algorithms. It was that every estimator answers the same four verbs, so a pipeline, a cross-validator and a grid search can be written once and work with all of them. That is what `insyra/ml` is for, and it is why v1 wraps rather than reimplements: the numerics are already here and already checked against R.

This change is the protocol itself and the wrapping. It is deliberately the smallest thing that is useful on its own — a caller can fit any model in `stats` through one interface and score new data with it — and it is what every later change depends on.

## What Changes
- Add `insyra/ml` with two one-method interfaces, `Transformer` and `Model`, and optional capabilities as separate interfaces
- Type `Transformer.Transform` to match `insyra.Scaler.Transform` and `insyra.Encoder.Transform` exactly, so all four scalers and all three encoders satisfy it with no adapter — verified: the assertions compile against the tree as it stands
- Add fitting functions for the models `stats` already has, each returning a `Model`
- Record on every fitted model which columns it was fit on, since nothing in `stats` does and positional binding is how a model silently scores the wrong columns
- Add a conformance test helper so a third-party model can check it obeys the contract

## Impact
- Affected specs: `ml-protocol`
- Affected code: new `ml/` package; no change to `stats` or the root package
- New package; nothing existing changes behaviour
