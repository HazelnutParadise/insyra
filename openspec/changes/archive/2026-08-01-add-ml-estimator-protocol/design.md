# Design: the `insyra/ml` protocol

This is the shape to implement. Where it departs from scikit-learn the reason is stated, because "we imitate scikit-learn" is the standing decision and every departure is an exception to it.

## What is being imitated, and what is not

scikit-learn's vocabulary and verbs, exactly: `Fit`, `Transform`, `FitTransform`, `Predict`, `PredictProba`, `Score`, `Pipeline`, and the estimator names themselves. A reader who knows scikit-learn should recognise this code.

Three things are not imitated, each because it rests on a Python feature Go does not have:

**Fitting returns a separate value.** scikit-learn mutates `self` and returns `self`, so an unfitted estimator is a runtime `NotFittedError`. Here fitting returns a distinct type, so using an unfitted model is a compile error. Spark ML, linfa and tidymodels all made the same move for the same reason.

**There is no `get_params` / `set_params` / `clone`.** scikit-learn builds all three on `inspect.signature(cls.__init__)`. Their two real uses are refitting per cross-validation fold and grid search. Both are served here by a fit *function*, described below.

**`Score` takes the metric as an argument.** scikit-learn's `ClassifierMixin.score` silently means accuracy and `RegressorMixin.score` silently means R². A hidden default that differs by estimator family is a trap; the metric is passed at the call site.

## The protocol

```go
// Package ml is Insyra's classical machine-learning layer.
package ml

import (
	"io"

	"github.com/HazelnutParadise/insyra"
)

// Transformer is a fitted preprocessing step. scikit-learn: transformer.transform(X).
//
// The signature is not a free choice. It is character-identical to
// insyra.Scaler.Transform (datatable_scale.go:38) and insyra.Encoder.Transform
// (datatable_encode.go:81), including the concrete *DataTable on both sides.
// That is what makes every existing scaler and encoder a legal pipeline step
// with no adapter and no change to the root package. Widening either side to
// IDataTable breaks that, which is the single largest reuse in the package.
type Transformer interface {
	Transform(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// Model is a fitted predictor. scikit-learn: predictor.predict(X).
type Model interface {
	// Features names the columns the model was fitted on, in the order it
	// expects them.
	//
	// This has no scikit-learn equivalent — it is `feature_names_in_`, which
	// sklearn added late and treats as advisory. Here it is part of the
	// interface because nothing in stats records it: LinearRegressionResult
	// (regression.go:55), GLMResult (regression_glm.go:21) and KMeansResult
	// (clustering.go:19) all omit it, and predictFromCoefficients validates only
	// the count (glm_predict.go:96). Positional-only binding is how a
	// correct-looking model silently scores the wrong columns.
	Features() []string

	// Predict scores new observations. Columns are matched by name against
	// Features, not by position.
	Predict(dt *insyra.DataTable) (*insyra.DataList, error)
}
```

Optional capabilities are separate interfaces, discovered by type assertion. That is the Go form of scikit-learn's `hasattr(est, "predict_proba")`, and it is how gonum splits `DerivativePredictor` out of `Predictor`. Adding a method to `Model` later would break every implementation; adding an interface here breaks nothing.

```go
// InverseTransformer is scikit-learn's inverse_transform. Every insyra.Scaler
// and every insyra.Encoder satisfies it today.
type InverseTransformer interface {
	InverseTransform(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// ProbaModel is scikit-learn's predict_proba. The contract, copied from
// sklearn: the column order of PredictProba is the row order of Classes. Go
// cannot enforce that, so the conformance check does.
type ProbaModel interface {
	Model
	Classes() *insyra.DataList
	PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)
}

// Importances is scikit-learn's feature_importances_, one value per entry of
// Features, same order.
type Importances interface {
	Model
	FeatureImportances() []float64
}

// Exporter writes the model in a form another runtime can read. Landed by a
// later change; declared here so implementations can be written against it.
type Exporter interface {
	Model
	ExportONNX(w io.Writer) error
}
```

## Fitting: a step is a function, not a configured object

This is the load-bearing departure, and it is what replaces `clone()`.

scikit-learn refits a step per cross-validation fold by cloning the *configured but unfitted* estimator object. Cloning requires reading the constructor's parameters back off the instance, which needs `inspect.signature`. Go has no equivalent that is not reflection over struct tags, which the repository uses nowhere.

A closure refits by being called again. Configuration is captured in it at the call site, in ordinary Go, checked by the compiler.

```go
// A Step is an unfitted preprocessing stage: a name and a way to fit one.
type Step struct {
	Name string
	Fit  func(x *insyra.DataTable, y *insyra.DataList) (Transformer, error)
}

// An Estimator is an unfitted model: a name and a way to fit one.
type Estimator struct {
	Name string
	Fit  func(x *insyra.DataTable, y *insyra.DataList) (Model, error)
}
```

A caller configures by closing over the options:

```go
est := ml.Estimator{
	Name: "ridge",
	Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
		return ml.FitLinearRegression(x, y)
	},
}
```

Cross-validation calls `est.Fit` once per fold. Grid search closes over each parameter combination. Neither needs to inspect anything.

## Wrapping what `stats` already has

Each wrapper is thin: call the `stats` function, record the column names, wrap the result. It must not reimplement any arithmetic — the point of wrapping is to inherit numerics already checked against R.

```go
func FitLinearRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error)
func FitPolynomialRegression(x *insyra.DataTable, y *insyra.DataList, degree int) (Model, error)
func FitLogisticRegression(x *insyra.DataTable, y *insyra.DataList, opts ...LogisticOptions) (ProbaModel, error)
func FitPoissonRegression(x *insyra.DataTable, y *insyra.DataList, opts ...PoissonOptions) (Model, error)
func FitGLM(x *insyra.DataTable, y *insyra.DataList, opts GLMOptions) (Model, error)
func FitKMeans(x *insyra.DataTable, k int, opts ...KMeansOptions) (Model, error)
func FitPCA(x *insyra.DataTable, components int) (Transformer, error)
func FitKNNClassifier(x *insyra.DataTable, y *insyra.DataList, k int, opts ...KNNOptions) (ProbaModel, error)
func FitKNNRegressor(x *insyra.DataTable, y *insyra.DataList, k int, opts ...KNNOptions) (Model, error)
```

`FitKMeans` returns a `Model` whose `Predict` is `KMeansResult.Assign` — that method exists as of this week, and it is the only selection-shaped operation in `stats`, which makes it the only one a device may ever accelerate by default.

`FitPCA` returns a `Transformer`, not a `Model`: PCA projects, it does not predict. Its `Transform` applies the `Center`, `Scale` and `Components` that `PCAResult` now carries.

The underlying result stays reachable, so wrapping costs a caller nothing:

```go
type LinearModel struct {
	Result   *stats.LinearRegressionResult
	features []string
}
```

## Options

Variadic options structs, zero value means default, arity guarded. This is the repository's idiom — `stats.KNNOptions` (knn.go:139), `stats.KMeansOptions` (clustering.go:13), `finance` options. Do not introduce functional options: `grep -rn "func With[A-Z]"` over the whole repository returns nothing, and `ml` should not be the one package that looks different.

```go
func FitLogisticRegression(x *insyra.DataTable, y *insyra.DataList, opts ...LogisticOptions) (ProbaModel, error) {
	if len(opts) > 1 {
		return nil, errors.New("ml: opts accepts at most one value")
	}
	...
}
```

## Errors

Returned, never logged and never stored on the instance. `stats` moved to error-first on 2026-04-08 and `ml` starts there. Do not use the root package's `Err()` pattern here.

## Column matching

`Predict` matches columns by name against `Features()`. A missing column, or a set that does not cover what the model needs, is an error naming what was missing. Extra columns are ignored — a caller scoring a table that also carries an id column should not have to strip it first.

This is the one place `ml` is stricter than `stats`, deliberately. `stats`' own `Predict` takes a variadic of columns and checks only the count, so passing them in the wrong order silently produces wrong numbers. `ml` exists partly to close that.

## Precision

Everything here reports `float64`, because everything here wraps `stats`. That is the reported-values row of the precision contract in `delivery-plan.md`, and no other row is engaged until a model computes its own arithmetic — the decision tree, in a later change.

A wrapped model must be bit-identical to calling the `stats` function directly. Not close: identical. The test asserts equality.

## The conformance check

```go
package mltest

// RunConformance exercises every rule the protocol states against a fitted
// model, so an implementation outside the package can check itself.
func RunConformance(t *testing.T, m ml.Model, x *insyra.DataTable, y *insyra.DataList)
```

What it checks: `Features()` is non-empty and has no duplicates; `Predict` on the training table returns one value per row; `Predict` with a renamed column errors; `Predict` with the columns reordered gives the same answer as before they were reordered — which is the property positional binding silently violates; and for a `ProbaModel`, that `PredictProba` has one column per entry of `Classes()`, in that order, with rows summing to one.

## What this change does not include

Pipelines, cross-validation, metrics, decision trees and ONNX export are each their own change. This one is the protocol and the wrapping, and it is useful on its own: a caller can fit anything in `stats` through one interface and score new data with it.
