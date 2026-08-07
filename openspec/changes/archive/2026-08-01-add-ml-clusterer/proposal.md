# Change: Let a clustering model say that its predictions are not measurements

## Why
`validateMetricModel` refuses a regression metric only when the model satisfies `Classifier`. A model that is neither a classifier nor a continuous predictor passes, and the metric then scores whatever numbers it produced.

The clearest case is in the package itself. `KMeansModel.Predict` returns cluster ids, and `KMeansModel` has no `Classes()` — correctly, because a cluster id is not a class from a known set. So it is not a `Classifier`, nothing refuses it, and:

```
CrossValidate(x, y, kmeansEstimator, 3, ml.RMSEMetric{})
  -> mean = 0.8047378541243649   err = <nil>
```

RMSE over cluster labels. Arithmetically correct, and meaningless.

The check cannot catch this as written. `Model.Predict` returns a `*insyra.DataList`, and there is no type-level difference between a predicted value and a cluster id — both are numbers in a list. The gap is not in the check but in what a model declares about itself.

The package already has the shape for this. `ProbaModel`, `Importances` and `Exporter` are optional interfaces discovered by assertion; a clustering model declaring itself the same way costs one interface and one method, and `validateMetricModel` gains one arm alongside the `Classifier` arm it already has.

## What Changes
- Add an optional interface a clustering model implements to say its predictions are group assignments rather than measurements
- Have the fitted KMeans model implement it, reporting how many clusters it converged on
- Refuse a regression metric against a model that declares itself a clusterer, the way one is already refused against a classifier
- Leave every other model untouched: nothing that does not implement the interface changes behaviour

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/interfaces.go`, `ml/models.go`, `ml/model_selection.go`
- Additive: one interface, one method on one model, one validation arm
- Closes https://github.com/HazelnutParadise/insyra/issues/193
