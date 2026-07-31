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
FitDecisionTreeClassifier
FitDecisionTreeRegressor
```

The regression, clustering, and KNN wrappers expose their underlying `stats` result through an exported `Result` field. The options types in `ml` are aliases of the corresponding `stats` options types.

Polynomial, exponential, and logarithmic regression require one feature. Logistic regression and KNN classification implement `ProbaModel`. Logistic probabilities use the fitted class order, and KNN probabilities use the class columns returned by `stats`.

KNN stores the training table because the underlying `stats.KNNClassify` and `stats.KNNRegress` functions accept training and test tables together. Predictions call those functions with the test table reordered by feature name.

## Decision trees

`FitDecisionTreeClassifier` and `FitDecisionTreeRegressor` fit deterministic
histogram trees. They return fitted models that implement `Model` and
`Importances`; the classifier also implements `ProbaModel`.

```go
tree, err := ml.FitDecisionTreeClassifier(trainX, trainY, ml.DecisionTreeOptions{
    MaxDepth:            5,
    MinSamplesLeaf:     2,
    CategoricalFeatures: []string{"region"},
})
predicted, err := tree.Predict(testX)
probabilities, err := tree.PredictProba(testX)
importance := tree.FeatureImportances()
```

Numeric features are converted to `float32` and binned with deterministic
type-7 quantile edges. Missing numeric values occupy their own bin. Categorical
features are split by a learned subset of categories, not by numeric encoding.
The default `MaxBins` is 32. `MaxDepth` and `MaxLeaves` use zero for unlimited;
`MinSamplesLeaf` defaults to 1.

At each split, missing values are sent to the branch with the better gain. A
tie, or a split that saw no missing values while fitting, defaults to the left
branch. An unseen category at scoring time uses that same stored default. Tied
gains resolve by feature order, then split order, then left for missing values,
so a fit is unchanged by row order. Classification classes are reported in a
deterministic order, and probability columns follow that order.

The tree accumulates classification counts and quantised regression sums in
integers. Leaf values and feature importances are reported as `float64`.
`Root` exposes the fitted nodes, and `LeafValues` returns leaf values in
left-to-right order.

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

## Pipelines

`NewPipeline` combines fitted preprocessing with a final `Estimator`. The
returned estimator refits every step on each call to `Fit`, and the fitted
pipeline implements `Model`.

```go
pipeline := ml.NewPipeline([]ml.Step{
    {
        Name: "scale numeric",
        Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Transformer, error) {
            scaler := insyra.NewStandardScaler()
            if err := scaler.Fit(x, "age", "income"); err != nil {
                return nil, err
            }
            return scaler, nil
        },
    },
}, ml.Estimator{
    Name: "linear",
    Fit: ml.FitLinearRegression,
})

model, err := pipeline.Fit(trainFeatures, target)
if err != nil {
    log.Fatal(err)
}
predictions, err := model.Predict(testFeatures)
```

Steps run in declaration order. A fitting or transformation error names the
step that failed. `model.Features()` reports the columns accepted at the raw
pipeline boundary. Since fitting functions create a fresh fitted transformer
when called, the same pipeline definition can be reused for cross-validation
or another training set without carrying parameters from an earlier fit.

`NewColumnTransformer` scopes a fitted transformer to named columns. The
remaining columns pass through unchanged, and columns with the same output
count retain their original positions. Root scalers and encoders already
support column selection during fitting, so they need no adapter.

```go
scoped := ml.NewColumnTransformer(fittedTransformer, "age", "income")
transformed, err := scoped.Transform(raw)
```

Fit preprocessing only on the training split. Fitting a scaler or encoder on
the complete table before splitting leaks information from the future test
rows into the fitted parameters. The resulting score can look better while
measuring a model that had already seen part of the data it was supposed to
hold out. Fit the pipeline on `train`, then call the fitted model on `test`.

## Conformance checks

The `ml/mltest` package checks an implementation's feature contract, name-based binding, missing and extra columns, reordered inputs, and probability output.

```go
mltest.RunConformance(t, model, features, target)
```
