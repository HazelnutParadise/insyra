# Change: Ridge and lasso — regularized regression for `stats` and `ml`

## Why
The regression family stops at unpenalized estimators. That leaves two ordinary situations without an answer. Collinear or near-collinear predictors make the OLS normal equations singular, so `LinearRegression` fails on data that ridge handles by design. And nothing in the library can produce a sparse model — lasso's ability to drive coefficients to exactly zero is the standard tool for feature selection in both applied work and papers. Ridge and lasso are the baseline expectation of any regression toolkit; their absence is the largest single gap between this library and one a practitioner would call standard.

Two conventions are settled up front:

**The objectives are scikit-learn's, stated exactly.** Ridge minimises `||y − Xβ||² + α·||β||²` and lasso minimises `(1/2n)·||y − Xβ||² + α·||β||₁`, both leaving the intercept unpenalized and neither standardising features. R's glmnet is the other candidate reference, but it standardises by default and scales its penalty differently, so "matches glmnet" and "matches sklearn" are different numbers from the same data. This library imitates scikit-learn's API; its objectives are the ones a caller comparing against sklearn will expect. The choice is recorded, and verification is against sklearn.

**No classical inference on penalized estimates.** The unpenalized results carry standard errors, t and p values. Penalized estimates have no exact classical sampling distribution, so reporting those fields would be fabricating precision. The result types deliberately do not have them.

## What Changes
- Add `stats.RidgeRegression` — closed form via the penalized normal equations, exact for any `α ≥ 0`, `α = 0` reproducing OLS
- Add `stats.LassoRegression` — coordinate descent with soft thresholding, sklearn-compatible objective, options for tolerance and iteration cap, convergence reported on the result
- Both accept multiple predictors, reuse the validated input loader (so unreadable values are refused with the row named), and carry `Predict` like the other regression results
- Verify both against scikit-learn through the cross-language baseline harness, behind a Python-only gate routed through the shared reference-toolchain switch
- Add `ml.FitRidgeRegression` and `ml.FitLassoRegression` wrapping them into the estimator protocol, conformance-checked
- Document both in `Docs/stats.md` and `Docs/ml.md`, including the objectives, the sklearn-not-glmnet decision, and the absence of inference fields

## Impact
- Affected specs: `stats-regression`, `ml-protocol`
- Affected code: new `stats/regression_regularized.go`, `stats/testdata/crosslang_baseline.py`, `ml/models.go`, docs, changelogs, `skills/insyra/`
- Additive; nothing existing changes behaviour
