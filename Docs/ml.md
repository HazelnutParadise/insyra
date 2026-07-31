# `ml` Package

The `ml` package provides one estimator protocol over the classical models already implemented by `stats`. It wraps fitted results without reimplementing the statistical algorithms.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/ml
```

## Protocol

`Model` exposes the fitted feature names and one prediction method. Prediction binds columns by name, in the order used during fitting. Missing features return an error naming the column. Extra columns are ignored.

```go
type Model interface {
    Features() []string
    Predict(*insyra.DataTable) (*insyra.DataList, error)
}

type Transformer interface {
    Transform(*insyra.DataTable) (*insyra.DataTable, error)
}
```

`ProbaModel`, `InverseTransformer`, `Importances`, and `Exporter` are optional capabilities. Discover them with a type assertion instead of assuming every model supports them.

`Step` and `Estimator` store fit functions. Configuration can be captured by a closure, so a caller can fit the same step repeatedly without cloning or reflection.

## Splitting and cross-validation

`KFold` returns disjoint folds. Rows are shuffled by default, and a seeded
`insyra.SamplingOptions` makes the assignment reproducible. Use
`PreserveOrder: true` when the source order is meaningful. `StratifiedKFold`
takes a label list and keeps each class represented in every fold; it returns
an error naming any class with fewer observations than `k`.

```go
folds, err := ml.StratifiedKFold(
    features,
    target,
    5,
    insyra.SamplingOptions{UseSeed: true, Seed: 42},
)
```

`CrossValidate` fits the supplied `Estimator` independently for every fold.
Classification metrics select stratified folds automatically. Put preprocessing
inside the estimator's `Fit` function so it is fitted only on that fold's
training data; this also makes a pipeline's preprocessing leakage-free.

```go
result, err := ml.CrossValidate(
    features,
    target,
    ml.Estimator{
        Name: "linear",
        Fit:  ml.FitLinearRegression,
    },
    5,
    ml.RMSEMetric{},
    insyra.SamplingOptions{UseSeed: true, Seed: 42},
)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Metric, result.Scores, result.Mean)
```

If fitting or scoring fails, the error names the one-based fold. The result
keeps every fold score rather than returning only the mean.

## Metrics

The metric is always supplied by the caller and its name is carried on the
result. Available metrics are:

| Metric | Model kind | Direct helper |
| --- | --- | --- |
| `AccuracyMetric` | classification labels | `Accuracy` |
| `LogLossMetric` | class probabilities | `LogLoss` |
| `ROCAUCMetric` | binary class probabilities | `ROCAUC` |
| `ConfusionMatrixMetric` | classification labels | `ConfusionMatrix` |
| `RMSEMetric` | regression | `RMSE` |
| `MAEMetric` | regression | `MAE` |
| `R2Metric` | regression | `R2` |

Classification models used with `CrossValidate` implement `Classifier` by
returning their class labels. Probability metrics additionally require
`ProbaModel`. A metric for the wrong model kind is rejected instead of
producing a meaningless score. Confusion-matrix cross-validation results are
available in `FoldResults`; their scalar `Scores` and `Mean` are `NaN`.

## Fitting models

```go
features := insyra.NewDataTable(
    insyra.NewDataList(1, 2, 3, 4).SetName("age"),
    insyra.NewDataList(10, 12, 15, 19).SetName("income"),
)
target := insyra.NewDataList(3, 5, 7, 9)

model, err := ml.FitLinearRegression(features, target)
if err != nil {
    log.Fatal(err)
}

scored, err := model.Predict(insyra.NewDataTable(
    insyra.NewDataList(20, 21).SetName("income"),
    insyra.NewDataList(5, 6).SetName("age"),
    insyra.NewDataList("row-a", "row-b").SetName("id"),
))
```

The available fitting functions are:

```go
FitLinearRegression
FitPolynomialRegression
FitExponentialRegression
FitLogarithmicRegression
FitLogisticRegression
FitPoissonRegression
FitGLM
FitKMeans
FitPCA
FitKNNClassifier
FitKNNRegressor
```

The regression, clustering, and KNN wrappers expose their underlying `stats` result through an exported `Result` field. The options types in `ml` are aliases of the corresponding `stats` options types.

Polynomial, exponential, and logarithmic regression require one feature. Logistic regression and KNN classification implement `ProbaModel`. Logistic probabilities use the fitted class order, and KNN probabilities use the class columns returned by `stats`.

KNN stores the training table because the underlying `stats.KNNClassify` and `stats.KNNRegress` functions accept training and test tables together. Predictions call those functions with the test table reordered by feature name.

## PCA transformation

`FitPCA` returns a `Transformer`. It applies the fitted `Center`, `Scale`, and component loadings to new tables and returns columns named `PC1`, `PC2`, and so on.

```go
transformer, err := ml.FitPCA(features, 2)
if err != nil {
    log.Fatal(err)
}
projected, err := transformer.Transform(features)
```

## Existing scalers and encoders

The root package's four scalers and three encoders already satisfy `ml.Transformer` and `ml.InverseTransformer`; no adapter is needed.

```go
scaler := insyra.NewStandardScaler()
if _, err := scaler.FitTransform(features, "age"); err != nil {
    log.Fatal(err)
}

var step ml.Transformer = scaler
scaled, err := step.Transform(features)
```

## Conformance checks

The `ml/mltest` package checks an implementation's feature contract, name-based binding, missing and extra columns, reordered inputs, and probability output.

```go
mltest.RunConformance(t, model, features, target)
```
