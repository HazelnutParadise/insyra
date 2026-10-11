# Design: ml-fit-returns-concrete-types

## The return types stay

The `Fit*` functions keep returning `Model`, `ProbaModel`, `Clusterer` or `Transformer`, as they do today, so every caller that compiles now still compiles. On a failed fit they return an untyped nil interface, not a nil pointer inside one. The closure case in the proposal therefore does not arise on this line; it arises only when a caller holds a concrete nil pointer, such as `(*ml.LinearModel)(nil)`, which is what the nil checks below cover.

## Nil receivers

The regression models' `Predict` checks `m == nil || m.Result == nil` at the top and returns an error naming the model kind, the wording the KNN and KMeans methods already use.

`ExportONNX(w, fitted)` checks `fitted == nil || isNilPointer(fitted)` before it dispatches on the model type. Every model's `ExportONNX(w)` method calls it, so one check covers them all.

The tree and ensemble models and the KNN and KMeans models already answer a nil receiver on the base commit. The port's tests pin that behaviour; no code changed for it.

## Why `Features` stays a value method

`Features` comes from the embedded `modelBase`, a value field, so calling it through a nil outer pointer dereferences before any method body runs. Declaring a nil-checking `Features` on each pointer type would shadow the promoted value method, and `ml.LinearModel{}` would stop satisfying an interface that names `Features`. The port tried that and reverted it, because v0.3.4 code relies on the value method set. Fixing nil `Features` therefore needs a different approach, which is on 0.4.

## Verification

- `wrapped_models_nil_test.go`: `Predict` on ten nil regression pointers returns an error without panicking; `ExportONNX` on five nil exporters returns an error and writes nothing; `PCATransformer` on nil returns nil `Features` and an error from `Transform`.
- `value_features_test.go`: each model value, including `ml.LinearModel{}`, satisfies an interface that names `Features`. This compile-time check guards against the pointer-method shadowing described above.
- `tree_models_nil_test.go`: the tree and ensemble models on nil return nil or an error and do not panic. These answers predate this change.
