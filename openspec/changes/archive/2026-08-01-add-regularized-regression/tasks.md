# Tasks

## 1. `stats`
- [x] 1.1 `RidgeRegression(y, alpha, xs...)` — penalized normal equations, intercept unpenalized, `Predict` on the result
- [x] 1.2 `LassoRegression(y, alpha, xs...)` with options for tolerance and iteration cap — coordinate descent with soft thresholding on centered data, convergence and iteration count on the result
- [x] 1.3 Both route input through the validated loader, and refuse a negative or non-finite penalty
- [x] 1.4 Neither result type carries standard errors, t or p values

## 2. Verification
- [x] 2.1 Ridge with zero penalty equals `LinearRegression`'s coefficients
- [x] 2.2 Ridge succeeds on collinear predictors that make `LinearRegression` fail
- [x] 2.3 Lasso produces exact zeros at a sufficient penalty, and reports non-convergence at a starved iteration cap
- [x] 2.4 Add `ridge` and `lasso` methods to the Python baseline script and compare coefficients, intercept and predictions against scikit-learn, behind a Python-only gate routed through the reference-toolchain switch
- [x] 2.5 Unreadable input is refused with the row named, inherited from the shared loader

## 3. `ml`
- [x] 3.1 `FitRidgeRegression` and `FitLassoRegression` wrapping the `stats` fits, with model types mirroring `LinearModel`
- [x] 3.2 Both pass `mltest.RunConformance`, and predictions through the wrapper equal the `stats` result's own

## 4. Documentation
- [x] 4.1 `Docs/stats.md`: both estimators, their exact objectives, the sklearn-not-glmnet decision, the missing-inference statement
- [x] 4.2 `Docs/ml.md`: the two fitting functions in the model table
- [x] 4.3 Update `skills/insyra/`
- [x] 4.4 Add entries to `CHANGELOG.md` and `CHANGELOG_TW.md`
