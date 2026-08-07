# Change: Let a caller-written metric say what input it needs

## Why
`ml.Metric` is exported and every method it requires is exported, so a caller can implement it. But the routing that decides *what a metric receives* keys off two unexported marker interfaces:

```go
type classLabelMetric interface{ needsClassLabels() bool }
type probabilityMetric interface{ needsProbabilities() bool }
```

A caller cannot implement an unexported method, so an external metric matches neither and always falls through to `model.Predict`. A metric needing class labels happens to work, since `Classifier.Predict` returns labels. A metric needing **probabilities** cannot be written at all.

That rules out a custom Brier score, a calibration measure, a multi-class log-loss variant, or any probability-based business metric — precisely the cases where someone writes their own rather than using a built-in.

Nothing errors. The metric receives a well-formed `Prediction` with `Probabilities` nil, so a caller who does not check computes a score from the wrong input and gets a number back. Demonstrated: a metric defined outside the package, cross-validated against a logistic model that *is* a `ProbaModel`, receives `Values=true Probabilities=false Classes=true`.

This is the mirror image of two defects already fixed here. `SimpleImputer` claimed an `InverseTransform` it could not honour; `LogisticModel.Predict` returned probabilities where `Classifier` promises labels. Both were a capability claimed but absent. This one is a capability present but unreachable — and the package already expresses optional capabilities as exported interfaces everywhere else, in `ProbaModel`, `Importances` and `Exporter`. The metric markers are the inconsistency.

## What Changes
- Export the two markers under the naming the package already uses for optional capabilities
- Keep the dispatch exactly as it is, so no built-in metric changes behaviour
- Document on `Prediction` which fields are populated when, since a nil `Probabilities` currently cannot be distinguished from a model that has none
- Verify with a metric defined outside the package, which is the only way to prove the extension point extends

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/model_selection.go`
- Additive: two interfaces and two method names become exported; built-in metrics keep their behaviour
- Closes https://github.com/HazelnutParadise/insyra/issues/192
