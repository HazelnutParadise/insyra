# Decision trees

Use the `ml` package for deterministic decision-tree classification and
regression over named `DataTable` columns.

```go
classifier, err := ml.FitDecisionTreeClassifier(
    trainX,
    trainY,
    ml.DecisionTreeOptions{
        MaxDepth:            4,
        CategoricalFeatures: []string{"region"},
    },
)
predicted, err := classifier.Predict(testX)
probabilities, err := classifier.PredictProba(testX)
```

Use `FitDecisionTreeRegressor` for continuous targets. `MaxDepth` and
`MaxLeaves` default to unlimited when zero. `MinSamplesLeaf` defaults to one,
and `MaxBins` defaults to 32. Numeric features use deterministic quantile
bins. Mark categorical columns by their fitted names in
`CategoricalFeatures`.

Missing values are not imputed. A split learns whether missing values go left
or right; ties and splits with no fitting-time missing values default left.
Unseen categories at scoring time use the same stored default. The classifier
reports sorted deterministic classes and probability columns in class order.

Both fitted trees implement `ml.Importances`. `FeatureImportances` returns one
`float64` per fitted feature, and `LeafValues` returns `float64` leaf values in
left-to-right order.
