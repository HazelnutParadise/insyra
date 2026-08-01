# Change: A metric says which way is better

## Why
`CrossValidationResult` reports a `Mean`, and nothing in the package says whether a larger one is good. For `accuracy`, `r2` and `roc_auc` it is; for `rmse`, `mae` and `log_loss` it is the opposite. A caller comparing two models by their means picks the worse one half the time, and the compiler cannot see it happen.

scikit-learn faced this and answered twice: `make_scorer(..., greater_is_better=...)`, and the `neg_mean_squared_error` naming convention that flips loss metrics so every scorer can be maximised. Both are the same admission — a score is uninterpretable without its direction, so the direction has to travel with it.

The direction cannot be inferred here. A metric supplied from outside this package could measure anything, and guessing from its name is the kind of inference this package has already ruled out: a model must declare what it predicts and a metric what it consumes, because a capability a caller cannot declare is a capability a caller does not have. Direction is the same kind of fact.

`ml` has never been released, so the metric protocol can still be given the method it should have had.

## What Changes
- Add a direction to the metric protocol, declared by every metric rather than inferred from its name
- Declare it on all seven built-in metrics; the confusion matrix declares that it has no direction, because it has no scalar score
- Add a comparison that answers "is this result better than that one" using the direction, so a caller selecting among models does not reimplement the sign
- Carry the direction on the cross-validation result, so a result that outlives the metric value that produced it is still interpretable
- Refuse a metric that declares no direction while returning a scalar score, rather than defaulting to one

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/model_selection.go`, `ml/mltest/`, `Docs/ml.md`, `skills/insyra/`
- **BREAKING** for a metric defined outside this package: it must declare a direction. `ml` is unreleased, so no released API changes.
