## Context

The `ml` protocol separates an unfitted `Estimator` from its fitted result, but
callers still need to apply preprocessing and a model in exactly the same order
on every later table. The pipeline is the boundary that owns that sequence and
prevents a scaler or encoder from being fitted once on data outside the
training partition.

The root package already supplies fitted scalers and encoders with compatible
`Transform` methods. The pipeline therefore belongs in `ml/` and stores the
fitted transformer values rather than introducing adapters or duplicating their
arithmetic.

## Goals / Non-Goals

**Goals:**

- Fit each transformer in order, transform the current table, then fit the
  final estimator on the transformed result.
- Refit every step from scratch each time the pipeline definition is fitted.
- Return a fitted object satisfying `Model` and preserve classifier,
  probability, and feature-importance capabilities from its final model.
- Canonicalize prediction inputs by fitted feature name before applying steps.
- Scope a transformer to named columns while passing all other columns through
  in their original positions.
- Name the step or estimator that failed.

**Non-Goals:**

- Adding estimator cloning, parameter grids, or parallel pipeline execution.
- Reimplementing scaler or encoder arithmetic in `ml`.
- Dropping or silently renaming input columns.
- Defining inverse transformation for lossy preprocessing.

## Decisions

### Use a reusable definition with a fit closure

`NewPipeline` copies the configured `Step` slice and returns an `Estimator`
whose `Fit` allocates a new fitted-step slice on every call. Each step's
`Fit` receives the current table and target, its fitted transformer immediately
transforms that table, and the final estimator receives the last result.

This follows the protocol's closure-based replacement for scikit-learn's
`clone`. Keeping fitted state inside the returned model prevents a second fit
from reusing statistics from the first. Mutating one shared pipeline instance
was rejected because it would make cross-validation leak state between folds.

### Preserve optional model capabilities with wrappers

The core fitted pipeline stores the feature list, fitted steps, and final
`Model`. `wrapFittedPipeline` chooses a small concrete wrapper based on the
final model's `Classifier`, `ProbaModel`, and `Importances` capabilities. These
wrappers forward `Classes`, `PredictProba`, and `FeatureImportances` while all
prediction paths run the same transformation sequence.

Adding these methods directly to `Model` was rejected because it would break
existing model implementations and force every estimator to support every
capability.

### Canonicalize feature order before transformation

The fitted pipeline records the input column names and `Predict` first builds
an ordered table from those names. This is required even for external steps
whose transformation is position-sensitive. Missing columns and duplicate or
empty fitted names are refused rather than allowing a reordered table to be
interpreted as a different feature set.

Extra prediction columns are ignored at the model boundary, consistent with
the `ml.Model` contract.

### Scope transforms by index while selecting by name

`ColumnTransformer` validates the requested names against the input table,
builds a selected table in the input's current order, and applies the wrapped
transformer only to that table. Pass-through columns are reinserted by their
original indices, not by names, so unnamed columns remain stable and duplicate
names cannot silently redirect a transform. A transformer that changes the
selected column count is inserted at the first selected position, while a
same-width transform replaces selected positions one-for-one.

Selecting by name provides a stable caller-facing API; reinserting by index
preserves the physical table layout. Rebuilding the output by a name map was
rejected because unnamed or repeated columns would be lost or reassigned.

### Fail closed and name every stage

Nil fit functions, nil fitted transformers, nil transformed tables, missing
columns, and transformer row-count changes return errors. Fit and transform
errors are wrapped with the generated step name or estimator name. The pipeline
does not log or store errors on the fitted object.

## Risks / Trade-offs

- **[Risk] A transformer changes the selected column count]** → Define the
  insertion rule explicitly, preserve row count, and reject ambiguous output
  only where the downstream model cannot resolve its fitted features.
- **[Risk] A downstream step expects a position that a scoped transform moves]**
  → Canonicalize the original feature order once and preserve pass-through
  indices through each scoped transform.
- **[Risk] Optional capability wrappers drift from the final model]** → Select
  wrappers only through type assertions and test every capability combination,
  including probabilities and importances together.
- **[Risk] A fitted pipeline is applied to a table with missing or duplicate
  names]** → Return a named validation error before any transformer runs.
- **[Risk] A transformer mutates its input table]** → Keep the pipeline contract
  on returned tables and test the root scalers and encoders as steps; later
  changes can add defensive cloning where a transformer requires it.

## Migration Plan

This is additive. Existing scalers, encoders, estimators, and direct model calls
continue to work. Callers construct a pipeline definition, fit it on training
data, and use the returned model for every later table. No persisted state or
data migration is needed; rollback removes the pipeline API and its docs.

## Open Questions

Pipeline serialization is intentionally left to the ONNX export change and
future persistence work. Inverse transformation for a composed pipeline also
needs an explicit policy for lossy steps and is not inferred here.
