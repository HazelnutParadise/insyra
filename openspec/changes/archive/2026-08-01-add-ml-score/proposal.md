# Change: `Score` — the one imitated verb that was never written

## Why
The estimator protocol's design names the scikit-learn verbs it imitates "exactly": `Fit`, `Transform`, `FitTransform`, `Predict`, `PredictProba`, `Score`, `Pipeline`. Six of the seven exist. `Score` does not.

The gap is not cosmetic. Scoring a fitted model on held-out data is the most ordinary thing a caller does after fitting one, and today the only route to it is `CrossValidate`, which refits. A caller who already has a fitted model and a test set has to reach for `Accuracy`, `RMSE` or `R2` directly — and in doing so takes on the work the harness already does correctly: asking the model for probabilities when the metric needs them, deriving class labels from an argmax when the model reports probabilities rather than labels, and refusing a model that cannot supply what the metric asked for.

That work is exactly where this package has already found three defects. Leaving every caller to redo it by hand outside the harness is leaving the same trap open.

## What Changes
- Add `Score`, which evaluates a fitted model on observations and their true values with a supplied metric
- Reuse the harness `CrossValidate` already uses, so a metric that needs probabilities or class labels is served identically whether it is reached through cross-validation or directly
- Refuse, before predicting, a model that cannot supply what the metric asked for — the same refusal `CrossValidate` makes
- Record the departure from scikit-learn: `score` there carries an implicit default metric attached to the estimator class, which Go cannot express; the metric is therefore an argument

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/model_selection.go`, `Docs/ml.md`, `skills/insyra/`
- Additive; nothing existing changes behaviour
