## Context

The `ml` protocol represents fitting as a closure on an `Estimator`. That
closure is enough to refit a model, but model selection still needs a stable
way to partition rows, preserve class balance, score the held-out rows, and
report failures without hiding which fold was affected.

The implementation belongs in `ml/` and reuses the root package's table and
sampling conventions. It must not mutate the caller's table or target list:
folds are views materialized as new tables and lists, while the estimator is
called afresh for every training fold.

## Goals / Non-Goals

**Goals:**

- Provide seeded ordinary and stratified k-fold table splitting.
- Ensure every row belongs to exactly one test fold and no row is lost.
- Refit an estimator, including all pipeline preprocessing, for every fold.
- Provide named classification and regression metrics with applicability
  checks.
- Return each fold's result, the metric name, and a scalar mean when one exists.

**Non-Goals:**

- Hyperparameter search, nested cross-validation, or estimator cloning.
- Parallel fold execution, which would change resource and failure semantics.
- Implicit accuracy or R² defaults based on model type.
- Adding a second table-sampling API unrelated to the root package's
  `SamplingOptions` conventions.

## Decisions

### Split by row indices, then rebuild tables

The splitter first produces disjoint row-index slices. `KFold` and
`StratifiedKFold` rebuild tables from those indices while preserving the input
column order and row values. Cross-validation derives the complementary
training indices for each held-out fold and builds the corresponding target
list.

This makes the partition property directly testable and avoids mutating or
reordering the caller's table. A splitter that returns only random tables was
rejected because it makes duplication and loss difficult to audit and does not
provide the training complement needed by cross-validation.

### Use seeded sampling options and deterministic validation

The implementation accepts one root `insyra.SamplingOptions` value. Its seed
and order settings control the row ordering, so repeating a split with the same
data and seed produces the same fold membership. Invalid option arity and
invalid values are rejected before any fold is returned.

Stratification groups row indices by stable label identity, distributes each
class across folds, and refuses a class with fewer observations than `k`. This
is preferable to silently producing an empty class in a test fold, which would
make probability metrics and classification scores misleading.

### Choose stratification from metric kind

`CrossValidate` selects stratified folds for `ClassificationMetric` and
ordinary folds for `RegressionMetric`. The caller still supplies the concrete
metric, and the metric's `Kind` is checked against the model capability before
evaluation. Classification metrics that need probabilities use `ProbaModel`;
label metrics use `Model` predictions.

Inferring a metric from the estimator family was rejected because wrappers and
pipelines can expose different capabilities from the final concrete type.

### Refit the estimator closure on every fold

For each fold, `CrossValidate` builds training and test data, calls
`estimator.Fit` once, validates the metric/model pairing, predicts on the held-
out table, and evaluates the supplied metric. A fitting, prediction, or score
error is wrapped with the one-based fold number.

This deliberately uses the protocol's fit closure instead of cloning an
estimator. It guarantees that a pipeline's transformers are fitted only on the
training partition and avoids reflection-based parameter extraction.

### Keep metric output explicit

Each metric implements `Name`, `Kind`, and `Evaluate`. Classification includes
accuracy, log loss, ROC AUC, and confusion matrix; regression includes RMSE,
MAE, and R². `CrossValidationResult` carries every fold's `MetricResult`, the
metric name, scalar scores, and their mean. Matrix-only results do not invent a
scalar mean.

The supplied metric name is copied onto every fold result so a custom evaluator
cannot make one fold disagree with the aggregate label.

## Risks / Trade-offs

- **[Risk] A small class cannot populate every stratified fold]** → Refuse the
  request with the class identity and required fold count instead of degrading
  to an unstratified split.
- **[Risk] Fold construction copies data and costs memory]** → Keep the API
  index-based internally and materialize only the train/test tables required by
  the current fold; correctness and isolation take priority over a new view
  abstraction.
- **[Risk] Probability metrics can be requested from a label-only model]** →
  Validate the `ProbaModel` capability before calling `PredictProba` and name
  the mismatch in the returned error.
- **[Risk] A fold can have no valid scalar score]** → Preserve the fold result
  and represent the aggregate mean as undefined rather than silently dropping
  the fold from the reported fold count.
- **[Risk] A custom metric returns an inconsistent name]** → Treat `Metric.Name`
  as authoritative and overwrite the per-fold display name after evaluation.

## Migration Plan

This is additive. Existing estimators and metrics are unchanged; callers opt
into `KFold`, `StratifiedKFold`, or `CrossValidate` and pass a metric explicitly.
No data migration is required. Removing the new file and documentation is a
normal source revert, with no change to existing model fitting.

## Open Questions

Parallel fold execution and hyperparameter search remain separate changes.
They should preserve the same fold partition and metric contracts before being
considered for inclusion.
