# Change: Let every regression in `stats` score new data, and check it against R

## Why
Four of the seven regression families cannot predict. `LinearRegressionResult`, `PolynomialRegressionResult`, `ExponentialRegressionResult` and `LogarithmicRegressionResult` carry coefficients and no way to use them. Only the GLM family — logistic, Poisson, GLM itself — has a `Predict`, which is the whole of it:

```
$ grep -h "^func ([a-z] \*\?[A-Z]" stats/*.go
LogisticRegressionResult.Predict        PoissonRegressionResult.Predict / PredictWithOffset
GLMResult.Predict / PredictWithOffset   ChiSquareTestResult.Show / FactorAnalysisResult.Show
```

Seven exported methods on fitted results, and five of them are prediction. A user who fits a line in Insyra today has to multiply the coefficients out by hand.

The second half is worse, because it is invisible. `stats` is validated against R — `stats/testdata/` holds 22 `.R` scripts and the tests shell out to `Rscript` and compare field by field. **None of those 22 scripts calls `predict()`.** The validation covers fitting. The prediction path, including the five methods that already exist, has never been checked against a reference implementation.

That matters now because the planned `insyra/ml` package is prediction-shaped: it wraps these results behind an estimator protocol whose central operation is scoring new data. Wrapping an unvalidated path and calling it validated would be the kind of claim this project has spent the last several days removing.

## What Changes
- Add `Predict` to `LinearRegressionResult`, `PolynomialRegressionResult`, `ExponentialRegressionResult` and `LogarithmicRegressionResult`
- Restate the model form each one predicts under, since the exponential and logarithmic fits are performed on transformed data and predicting means undoing that transform
- Add R reference scripts calling `predict()` for all seven families, and compare against them the way the existing cross-language tests do
- Record what R's `predict()` does that Insyra's does not, rather than silently differing — R returns fitted values, standard errors and intervals; matching the point estimate is this change's scope and the rest is stated as a known gap

## Impact
- Affected specs: `stats-regression`
- Affected code: `stats/regression.go`, `stats/regression_shared.go`, `stats/testdata/`, and the cross-language regression tests
- Additive: no existing signature changes, and the five existing `Predict` methods keep their behaviour — this change validates them, it does not alter them
