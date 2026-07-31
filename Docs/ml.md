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

Pipelines, cross-validation, metrics, decision trees, and ONNX export are separate concerns and are not included in this package's first protocol.
