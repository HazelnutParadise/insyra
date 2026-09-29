# Proposal: ml-one-onnx-export-name

## Why

ML-2 of the API review ([#264](https://github.com/HazelnutParadise/insyra/issues/264)) found two names for two things in `ml`:

- `ExportONNX(w io.Writer, fitted any)` and `WriteONNX(w io.Writer, fitted any)` do the same thing; `WriteONNX` calls `ExportONNX`. Both take `any`, so a value that is not a model at all, a `*PCATransformer` or a `string`, compiles and is refused only at run time.
- `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` are type aliases of `DecisionTreeOptions`, which both tree functions take.

The owner ruled on #211 that each function has one name, and that an old name stays for one release as a Deprecated wrapper or alias that keeps its old meaning.

The finding also lists the `Accuracy()`-style helpers beside the `AccuracyMetric{}` types and the variadic `opts ...Options` of the fit functions. The helpers are a convenience over the metric types, not a second name for one function: one computes a score from two lists, the other is a metric value that cross-validation, grid search and `Score` take. They stay. The variadic options follow the settings rule the owner set on #213 (an optional trailing `opts ...XxxOptions`, more than one value an error), so they stay as well.

## What Changes

- **BREAKING**: `ExportONNX(w io.Writer, fitted Model) error`. Every model the exporter supports, fitted pipelines included, is a `Model`, so passing one is unchanged; passing anything else stops compiling instead of failing at run time. A nil `Model` is an error.
- `WriteONNX` is **Deprecated** in favour of `ExportONNX`. It keeps its `any` parameter and its meaning for one release: a `Model` is exported exactly as `ExportONNX` exports it, and anything else is refused with the error it gave before.
- `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` are **Deprecated** in favour of `DecisionTreeOptions`. They stay aliases, so they are the same type.
- The removals are recorded as an `AGENTS.md` follow-up for the release after the one that ships this.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ml-onnx`: adds "One export function, typed by the protocol".
- `ml-trees`: adds "One options type for trees".

## Impact

- `ml/onnx_export.go`, `ml/decision_tree.go` and their tests.
- `Docs/ml.md`, both changelogs, `AGENTS.md` (removal follow-up), `api-review.md` (ML-2), `delivery-status.md`.
