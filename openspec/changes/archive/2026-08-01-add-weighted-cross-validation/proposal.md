# Change: A weights channel for cross-validation, without breaking the protocol

## Why
`FitWeightedLinearRegression` landed with its limitation stated: weights do not flow through `CrossValidate`, because folds subset rows and nothing subsets the weights with them. A caller who captures weights in a fit closure gets exactly that misalignment, silently — the weight of row 40 applied to whatever row the fold shuffle put in position 40.

The feared protocol break turns out not to be needed. scikit-learn routes `sample_weight` to *fit* via `fit_params`; its scorers stay unweighted unless separately asked. Imitating that means the `Metric` protocol is untouched, and the estimator side needs only an optional second fitting function — existing estimators, pipelines and grid searches compile and behave unchanged.

The tree-weights question stays open and is not touched here: float weights in the histogram accumulators would break the fixed-point precision contract, and that is an architecture decision recorded in `delivery-status.md`, not a feature to smuggle in.

## What Changes
- Add optional `FitWeighted` to `Estimator` — same shape as `Fit` plus a weights list; nil means the estimator does not accept weights
- Add `CrossValidateWeighted(x, y, weights, estimator, k, metric, options...)`: validates weights on the same terms WLS does (strictly positive, finite, one per row), subsets them with each fold's training rows, and fits through `FitWeighted`
- Held-out scoring stays unweighted, as scikit-learn's default does; stated in the documentation rather than discovered
- Refuse an estimator without `FitWeighted`, rather than falling back to unweighted fitting and reporting a weighted result that is not one
- Update the `FitWeightedLinearRegression` documentation and skill guidance: the limitation this closes is closed

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/interfaces.go`, `ml/model_selection.go`, docs, changelogs, `skills/insyra/`
- Additive; `Estimator` gains an optional field, nothing existing changes behaviour
