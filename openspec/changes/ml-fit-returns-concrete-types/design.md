# Design: ml-fit-returns-concrete-types

## The seam keeps its interface

`Estimator.Fit` is `func(x, y) (Model, error)` and `Step.Fit` is `func(x, y) (Transformer, error)`. They stay. A pipeline, a grid search and cross-validation hold models of any family through them, and `ENG.md` names the fit closure as the package's test seam. Only the `Fit*` functions change, and they are the ones a caller calls directly.

Go does not convert function types, so `Estimator{Fit: ml.FitLinearRegression}` stops compiling. It was already the only form that worked for three functions (`FitLinearRegression`, `FitExponentialRegression`, `FitLogarithmicRegression`); every other `Fit*` function takes an extra argument or a variadic options value and already needed a closure, the tree functions included. A generic adapter was considered and left out: it would add a name to save one line in three cases.

## Nil receivers

A closure `return ml.FitLinearRegression(x, y)` declared to return `(ml.Model, error)` converts a nil `*LinearModel` into a non-nil `Model` when the fit fails. The package's own callers (`CrossValidate`, `CrossValidateWeighted`, `GridSearch`, pipelines) check the error first and are not affected; `Score` already refuses a nil pointer through `isNilPointer`. A caller who checks the model instead of the error reaches methods on a nil receiver, which today panic for most types.

`Features` comes from the embedded `modelBase`, a value field, so calling it through a nil outer pointer dereferences before any method body runs. Each returned type therefore gets its own `Features` that checks for nil and calls the embedded one, the way `PCATransformer` already does. `Predict` and the other methods check for nil at the top and return an error naming the model kind, the wording the KNN and KMeans methods already use (`ml: kmeans model is nil`).

`ExportONNX(w, fitted)` checks for a nil pointer with the existing `isNilPointer` before it dispatches, so every model's `ExportONNX(w)` method, which calls it, is covered by one check.

## Verification

- A test assigns the result of each of the fourteen wrapping `Fit*` functions to a variable of its concrete type, reads `Result` from it, and puts it in a slice of the interface the function used to return. Against the old code it does not compile. The six tree and ensemble functions already returned their own types.
- Two tests call every exported method of the twenty returned types on a nil receiver, one for the fourteen wrapped models and one for the six tree and ensemble types, and require no panic and an error or empty value. Against the old code both panic.
- The existing wrapper-parity tests (`ml/models_test.go`) keep proving that the wrapped numbers are unchanged.
