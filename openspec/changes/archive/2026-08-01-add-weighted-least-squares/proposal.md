# Change: Weighted least squares — the sample-weight slice with exact inference

## Why
Sample weights were requested across the library. One family can have them exactly and verifiably today: weighted least squares, where the weighted normal equations, the coefficient covariance and the classical inference are all closed-form and checkable against statsmodels' `WLS` to numerical precision.

The rest of the request is deliberately not in this change, because it collides with two standing designs. Tree sample weights would push float weights into the histogram accumulators, which the precision contract fixed as integers for associativity — changing that is an architecture decision, not a feature. And weights flowing through cross-validation need a channel in `Estimator.Fit(x, y)` and `Metric.Evaluate`, which is a protocol change. Both are surfaced for a decision rather than smuggled in.

## What Changes
- Add `stats.WeightedLinearRegression(y, weights, xs...)`: weighted normal equations, full classical inference (standard errors, t and p values), weighted R², residuals and `Predict`
- Weights must be strictly positive finite numbers, one per observation — zero-weight exclusion semantics are ambiguous across references and are refused rather than guessed
- Verify coefficients, standard errors, t and p values, weighted R² and predictions against statsmodels' `WLS`, behind the Python reference gate
- Add `ml.FitWeightedLinearRegression(x, y, weights)`, documented plainly: usable with `Fit`, `Predict` and `Score`; weights do **not** flow through `CrossValidate`, which has no weights channel — that is the recorded protocol limitation
- Uniform weights reproduce ordinary least squares exactly

## Impact
- Affected specs: `stats-regression`
- Affected code: new `stats/regression_weighted.go`, `stats/testdata/crosslang_baseline.py`, `ml/models.go`, docs, changelogs, `skills/insyra/`
- Additive; nothing existing changes behaviour
