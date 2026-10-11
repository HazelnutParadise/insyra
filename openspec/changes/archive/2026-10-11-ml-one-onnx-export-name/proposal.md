# Proposal: ml-one-onnx-export-name

## Why

ML-2 of the API review ([#264](https://github.com/HazelnutParadise/insyra/issues/264)) found two names for two things in `ml`:

- `ExportONNX(w io.Writer, fitted any)` and `WriteONNX(w io.Writer, fitted any)` do the same thing; `WriteONNX` calls `ExportONNX`. Both take `any`, so a value that is not a model at all, a `*PCATransformer` or a `string`, compiles and is refused only at run time.
- `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` are type aliases of `DecisionTreeOptions`, which both tree functions take.

The owner ruled on #211 that each function has one name, and that an old name stays for one release as a Deprecated wrapper or alias that keeps its old meaning.

The finding also lists the `Accuracy()`-style helpers beside the `AccuracyMetric{}` types and the variadic `opts ...Options` of the fit functions. The helpers are a convenience over the metric types, not a second name for one function: one computes a score from two lists, the other is a metric value that cross-validation, grid search and `Score` take. They stay. The variadic options follow the settings rule the owner set on #213 (an optional trailing `opts ...XxxOptions`, more than one value an error), so they stay as well.

## What Changes

- `WriteONNX` is **Deprecated** in favour of `ExportONNX`. It keeps its `any` parameter and its meaning for one release: it exports exactly what `ExportONNX` exports and refuses exactly what `ExportONNX` refuses. It is removed in 0.4.
- `ExportONNX` keeps its `any` parameter on this line.
- `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` are **Deprecated** in favour of `DecisionTreeOptions`. They stay aliases, so they are the same type. They are removed in 0.4.
- Left on 0.4: **BREAKING** `ExportONNX(w io.Writer, fitted Model) error`. A caller holding an `any` value still compiles on this line; a `Model` parameter would stop it compiling, which the 0.3.x line does not take.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ml-onnx`: adds "WriteONNX is a Deprecated synonym of ExportONNX" and "ExportONNX keeps its any parameter".
- `ml-trees`: adds "One options type for trees".

## Impact

- `ml/onnx_export.go`, `ml/decision_tree.go` and their tests.
- `Docs/ml.md`, `CHANGELOG.md` and `CHANGELOG_TW.md`.
