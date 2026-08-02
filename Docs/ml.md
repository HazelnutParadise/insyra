# `ml` Package

The `ml` package provides one estimator protocol over the classical models already implemented by `stats`. It wraps fitted results without reimplementing the statistical algorithms.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/ml
```

## Protocol

`Model` exposes the fitted feature names and one prediction method. Fitted
feature columns must have unique, non-empty names. Prediction binds columns by
name, in the order used during fitting. Missing features return an error naming
the column. Extra columns are ignored.

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

| Metric | Model kind | Better | Direct helper |
| --- | --- | --- | --- |
| `AccuracyMetric` | classification labels | higher | `Accuracy` |
| `LogLossMetric` | class probabilities | lower | `LogLoss` |
| `ROCAUCMetric` | binary class probabilities | higher | `ROCAUC` |
| `PrecisionMetric` | classification labels | higher | `Precision` |
| `RecallMetric` | classification labels | higher | `Recall` |
| `F1Metric` | classification labels | higher | `F1` |
| `ConfusionMatrixMetric` | classification labels | — | `ConfusionMatrix` |
| `RMSEMetric` | regression | lower | `RMSE` |
| `MAEMetric` | regression | lower | `MAE` |
| `R2Metric` | regression | higher | `R2` |

Classification models used with `CrossValidate` implement `Classifier` by
returning their class labels. Probability metrics additionally require
`ProbaModel`. A metric for the wrong model kind is rejected instead of
producing a meaningless score. Confusion-matrix cross-validation results are
available in `FoldResults`; their scalar `Scores` and `Mean` are `NaN`.

### Precision, recall and F1 averaging

The three per-class metrics take a `ClassAverage`:

| Average | Meaning |
| --- | --- |
| `MacroAverage` (default) | unweighted mean over every observed class |
| `MicroAverage` | counts combined before dividing — equal to accuracy for single-label classification |
| `WeightedAverage` | per-class scores weighted by how often each class actually occurs |
| `BinaryAverage` | one named class alone; `PositiveClass` is required |

```go
score, err := ml.Score(model, x, y, ml.F1Metric{
    Average:       ml.BinaryAverage,
    PositiveClass: "churned",
})
```

Two deliberate departures from scikit-learn. Its default is binary averaging
with `pos_label=1`, which works there because its labels are usually the
integers 0 and 1; over arbitrary labels that default is a guess, so the default
here is macro. And the positive class is never chosen for you: unlike ROC AUC —
which is invariant under swapping the class — precision and recall of the two
classes are different numbers about different mistakes, so binary averaging
refuses to run without one named. A positive class combined with any other
average is refused rather than silently ignored. A class that never appears in
the predictions contributes precision 0, matching scikit-learn's
`zero_division=0`; the direct helpers `Precision`, `Recall` and `F1` compute
the macro default.

### Weighted cross-validation

An estimator declares weight support by setting the optional `FitWeighted`
field; everything that does not mention weights is unchanged. Held-out scoring
stays **unweighted**, matching scikit-learn's `cross_validate` default —
sample weights say how much each observation influences the *fit*, not what
the evaluation metric means. An estimator without `FitWeighted` is refused
rather than silently fitted unweighted.

```go
result, err := ml.CrossValidateWeighted(x, y, weights, ml.Estimator{
    Name: "wls",
    FitWeighted: func(x *insyra.DataTable, y *insyra.DataList, w *insyra.DataList) (ml.Model, error) {
        return ml.FitWeightedLinearRegression(x, y, w)
    },
}, 5, ml.RMSEMetric{})
```

### Grid search

`GridSearch` cross-validates every candidate on identical folds and returns the
winner refitted on the full data:

```go
result, err := ml.GridSearch(features, target, []ml.Estimator{
    {Name: "ridge a=0.1", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
        return ml.FitRidgeRegression(x, y, 0.1)
    }},
    {Name: "ridge a=1.0", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
        return ml.FitRidgeRegression(x, y, 1.0)
    }},
}, 5, ml.RMSEMetric{})
// result.BestName, result.BestModel (refit on all rows), result.Results, result.Seed
```

scikit-learn's `GridSearchCV` expands a parameter grid by reflecting over
constructor parameters — the machinery `clone()` needed, which this protocol
does not have. The grid therefore arrives already expanded: a slice of named
estimators, names required and unique. What `GridSearch` centralises is the
part that goes silently wrong by hand: every candidate is scored on identical
folds (a seed is drawn once and reported on the result when none is supplied),
ranking follows the metric's declared direction, ties keep the earliest
candidate, and the winner is refitted on all rows rather than a fold subset.

### Which score is better

Half the metrics improve as their score rises and half as it falls, so a `Mean`
on its own cannot be ranked. Every metric declares its direction, the result
carries it, and `Better` uses it:

```go
withTree, _ := ml.CrossValidate(x, y, treePipeline, 5, ml.RMSEMetric{})
withLinear, _ := ml.CrossValidate(x, y, linearPipeline, 5, ml.RMSEMetric{})

treeWins, err := ml.Better(withTree, withLinear)  // smaller RMSE wins
```

`Better` refuses two results from different metrics, and refuses a metric that
declares `NoDirection` — the confusion matrix, whose result is not a scalar.

`ROCAUCMetric` treats the second of the model's classes as positive, ordered by
the sorted distinct training labels. Which one that is does not affect the
score: the two probability columns are complementary, so naming the other class
would swap both the positive label and the score column, and the two swaps
cancel exactly.

### Scoring a fitted model

`Score` evaluates a model you already hold, without fitting anything:

```go
result, err := ml.Score(model, testFeatures, testTarget, ml.RMSEMetric{})
```

It runs the same compatibility check and the same prediction assembly
`CrossValidate` runs, so a metric that needs probabilities — or needs class
labels from a model that reports probabilities — is served identically either
way. scikit-learn's `score` carries a default metric on the estimator class;
Go has nowhere to hang that, so the metric is an argument.

### Writing your own metric

`Metric` is four methods, and a metric written outside this package works the
same way the built-in ones do:

```go
type BrierScore struct{}

func (BrierScore) Name() string        { return "brier" }
func (BrierScore) Kind() ml.MetricKind { return ml.ClassificationMetric }

// A smaller Brier score is a better one. Every metric must say, because a
// score whose direction nobody knows cannot be acted on — and it cannot be
// guessed from the name.
func (BrierScore) Direction() ml.MetricDirection { return ml.LowerIsBetter }

// Declaring this is what makes probabilities arrive. Without it the metric
// receives the model's predictions instead.
func (BrierScore) NeedsProbabilities() bool { return true }

func (BrierScore) Evaluate(yTrue *insyra.DataList, p ml.Prediction) (ml.MetricResult, error) {
    // p.Probabilities holds one column per class, named and ordered by p.Classes.
    ...
}
```

A model says what kind of thing it predicts by implementing an optional
interface. `Classifier` means the predictions are classes from a known set;
`Clusterer` means they are group assignments discovered by the fit. A model
implementing neither predicts measurements. The distinction is what stops a
regression metric scoring cluster ids — `Predict` returns a `DataList` either
way, so nothing else can tell them apart.

A metric says what input it needs by implementing one of two interfaces:

| Interface | What arrives | Requirement on the model |
| --- | --- | --- |
| `ProbabilityMetric` | `Prediction.Probabilities` with `Prediction.Classes` naming its columns | must implement `ProbaModel`, or the run is refused before fitting |
| `ClassLabelMetric` | `Prediction.Values` holding class labels | none; labels come from `Predict`, or from the argmax of the probabilities when the model reports them |
| neither | `Prediction.Values` holding the model's own `Predict` output | none |

Both interfaces are read, not merely detected: a metric that implements one and
answers `false` is treated exactly as one that did not implement it. That lets a
metric decide at runtime what it needs.

Which fields of `Prediction` are populated follows from that table alone. A nil
`Probabilities` means the metric did not ask for them — never that the model
could not supply them, since a model that cannot is refused first.

## Fitting models

### Values that are not numbers

The wrapped `stats` families refuse a value they cannot read as a finite number
rather than substituting one; see [the table in the `stats`
documentation](/Docs/stats.md#values-that-are-not-numbers). Decision trees are
the exception and are deliberately different: a missing **feature** value is
kept, and the tree learns per node which way such rows should go, which is the
standard treatment for that family. A missing **target** is refused, because
there is no direction to learn for the thing being predicted.


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
FitWeightedLinearRegression
FitRidgeRegression
FitLassoRegression
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
FitRandomForestClassifier
FitRandomForestRegressor
FitGradientBoostingClassifier
FitGradientBoostingRegressor
```

The regression, clustering, and KNN wrappers expose their underlying `stats` result through an exported `Result` field. The options types in `ml` are aliases of the corresponding `stats` options types.

`FitWeightedLinearRegression` takes one strictly positive weight per training
row. To cross-validate it, use `CrossValidateWeighted`, which subsets the
weights with each fold's training rows so alignment holds by construction —
a fit closure capturing the full weight list cannot provide that, because
folds shuffle and subset rows. Ridge and
lasso take the penalty strength as a third argument, using
scikit-learn's objectives exactly — see [the `stats`
documentation](/Docs/stats.md#ridge-regression) for the definitions and for why
the penalized results carry no standard errors. Polynomial, exponential, and
logarithmic regression require one feature. Logistic regression and KNN classification implement `ProbaModel`; logistic `Predict` returns class labels and `PredictProba` returns response probabilities. Logistic probabilities use the fitted class order, and KNN probabilities use the class columns returned by `stats`. The `ml` wrappers reject Poisson or GLM offsets because `Model.Predict` has no input for a new row-wise offset.

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
integers. Regression impurity is mean squared error, and the fixed-point scale
expands for small target magnitudes within the overflow bound. Leaf values and
feature importances are reported as `float64`.
`Root` exposes the fitted nodes, and `LeafValues` returns leaf values in
left-to-right order.

### Split styles

Numeric splits come in both mainstream styles:

| Style | How | When |
| --- | --- | --- |
| Histogram (default) | quantile bins, at most `MaxBins` candidates per feature | large data; LightGBM's design |
| `ExactSplits: true` | every midpoint between adjacent distinct values | scikit-learn's CART; verified prediction-for-prediction against it |

Exact splitting costs O(distinct values) candidates per feature per node
against the histogram's O(MaxBins), which is why histogram stays the default.
Combining `ExactSplits` with `MaxBins` is refused. Ensembles inherit the choice
through their `Tree` options. One caveat inherent to the comparison: on nodes
so small that several splits tie exactly, scikit-learn breaks the tie with a
per-node random feature order, so deep trees on tiny nodes can legitimately
differ between any two implementations.

## Ensembles

Both ensemble families are built from the same histogram trees.

**Random forests** (`FitRandomForestClassifier`, `FitRandomForestRegressor`)
reduce variance: each tree grows on a bootstrap resample with each split
restricted to a random feature subset (scikit-learn's defaults — 100 trees, √p
features for classification, all p for regression). Classification averages the
trees' probabilities and answers the largest; regression averages predictions.
Trees fit in parallel, but every draw derives from one forest seed, so the same
seed always gives the same forest — an unseeded fit draws one and reports it as
`Seed` on the model. Because classes are collected from the full target before
any resampling, every tree scores over one shared class order even when its
bootstrap sample misses a class.

```go
seed := int64(42)
forest, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{
    Trees: 200, Seed: &seed,
})
```

**Gradient boosting** (`FitGradientBoostingRegressor`,
`FitGradientBoostingClassifier`) reduces bias: each stage fits a depth-3 tree
to what the previous stages left unexplained, shrunk by the learning rate
(defaults 100 stages, rate 0.1). Regression boosts squared loss, where the
tree's own leaf means are already optimal. Binary classification boosts
logistic loss, replacing each leaf's value with the Newton step
`Σ(y−p)/Σp(1−p)` so the additive log-odds model converges on the loss actually
being boosted. Boosting is deterministic — no seed to manage. Fitting stops
early when the residuals reach zero, and `Stages` on the model reports how many
rounds ran. **Multiclass boosting is refused** in this version rather than
approximated; use a random forest for more than two classes.

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
pipeline boundary. The fitted pipeline retains classifier, probability, and
feature-importance capabilities from its final model. Raw inputs are reordered
to the fitted feature order before steps run, so position-sensitive steps see
the same schema at prediction time. Since fitting functions create a fresh
fitted transformer when called, the same pipeline definition can be reused for
cross-validation or another training set without carrying parameters from an
earlier fit.

`NewColumnTransformer` scopes a fitted transformer to named columns. The
remaining columns pass through unchanged by their original positions,
including unnamed columns. The selected columns must have names. Columns with
the same output count retain their original positions. Root scalers and
encoders already support column selection during fitting, so they need no
adapter.

```go
scoped := ml.NewColumnTransformer(fittedTransformer, "age", "income")
transformed, err := scoped.Transform(raw)
```


### The columns the estimator actually saw

`Features()` names the columns a pipeline accepts. When a step changes the
column count — a one-hot encoder is the ordinary case — the final estimator was
fitted on different columns, and `TransformedFeatureNames` is what names those:

```go
if expanded, ok := model.(ml.TransformedFeatures); ok {
    names := expanded.TransformedFeatureNames()
    importances := model.(ml.Importances).FeatureImportances()
    // one importance per name, in the same order
}
```

**Feature importances reported by a pipeline are indexed by these names, not by
`Features()`.** A pipeline over two columns that encodes one of them into three
reports two feature names and four importances; reading them together without
`TransformedFeatureNames` attributes every number to the wrong column.

### Leakage

Cross-validating a pipeline is leakage-free by construction, not by convention:
`CrossValidate` calls the pipeline's `Fit` once per fold with that fold's
training rows, and `Fit` refits every step. A step never sees a held-out row
before it is scored. The property that has to hold for this is that
preprocessing lives *inside* the pipeline; a transformer fitted on the whole
dataset and then passed in already-fitted has seen everything.

## ONNX export

Export a fitted model without a C dependency. The exporter writes a standard
ONNX `ModelProto` directly to an `io.Writer`:

```go
var modelFile bytes.Buffer
if err := ml.ExportONNX(&modelFile, model); err != nil {
    log.Fatal(err)
}
```

`LinearModel`, `RidgeModel`, `LassoModel`, `WeightedLinearModel`,
`LogisticModel`, `DecisionTreeClassifier`, `DecisionTreeRegressor`,
`RandomForestClassifier`, `RandomForestRegressor`,
`GradientBoostingRegressor`, `GradientBoostingClassifier`, and fitted
pipelines containing supported root scalers or encoders implement
`ml.Exporter`. Ensembles export as one multi-tree `TreeEnsemble`: a forest's
leaf contributions are scaled by 1/T so the runtime's sum is the forest's
average, and boosting bakes the learning rate into the leaf weights with the
prior as the ensemble's base value — the export is the model, not an
approximation of it. The export tests execute the generated graph in
`onnxruntime`, rather than accepting it by construction. The loop also closes inside the
library: [`dl`](/Docs/dl.md) loads every exported family — including pipelines
— and reproduces the fitted model's own predictions in pure Go. A pipeline is exported as one
graph, so the ONNX runtime receives the raw feature columns rather than a
pre-transformed table. ONNX stores model attributes as `float32`; predictions
are compared with the tolerance of that exchange format. The binary
`LinearClassifier` export is a known interoperability edge: the exporter writes
one score with `post_transform=LOGISTIC`, while the current `onnxruntime`
reference returns the raw score and its complement. The `dl` closure test keeps
the `ml` model, `dl`, and reference outputs separate so this mismatch cannot be
mistaken for a passing probability round trip.

The exporter refuses polynomial, exponential, logarithmic, Poisson, GLM,
KMeans, KNN, PCA, fitted imputers, and custom transformers. These models do
not have a faithful representation in this change. Encoder configurations with
`UnknownError`, `UnknownAsNew`, or a fitted nil category are also refused,
because ONNX cannot represent those semantics without changing predictions.
Refusal happens before the writer is touched. ONNX import is not part of the
`ml` package.

The test suite round-trips exportable models and pipelines through
`onnxruntime` when `python3` and that runtime are installed. When the runtime
is unavailable, the test reports an explicit skip rather than treating the
independent verification as passed.

Fit preprocessing only on the training split. Fitting a scaler or encoder on
the complete table before splitting leaks information from the future test
rows into the fitted parameters. The resulting score can look better while
measuring a model that had already seen part of the data it was supposed to
hold out. Fit the pipeline on `train`, then call the fitted model on `test`.

## Conformance checks

The `ml/mltest` package checks an implementation's feature contract, name-based binding, missing and extra columns, reordered inputs, and probability output.

For a model that reports probabilities, the class order is checked by value and
not only by column name: for every row, the class `Predict` returns must be the
class whose probability column holds that row's largest probability. Ties are
allowed. A model that picks its class by some other rule, such as a decision
threshold that is not the largest probability, will not pass this check.

```go
mltest.RunConformance(t, model, features, target)
```
