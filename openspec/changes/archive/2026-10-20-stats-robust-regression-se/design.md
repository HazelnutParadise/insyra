# Design: stats-robust-regression-se

## Where the code goes

`stats/AGENTS.md` keeps private regression math in `olsutil.go` (layer 3). The sandwich covariance (`olsSandwichCov`) and the inference helper that takes per-coefficient variances live there; `regression.go` only holds the public types and wiring. `computeCoeffInference` now builds the classical variances `mse * (X'X)^-1[i][i]` and hands them to the same helper, so the default path performs the same floating-point operations as before.

## One code path for HC0, HC1 and HAC

Every estimator is `B M B` with `B = (X'X)^-1` and `M` built from scaled scores `u_t = c_t x_t`:

| CovType | c_t | M |
| --- | --- | --- |
| HC0 | e_t | Σ u_t u_t' |
| HC1 | e_t | n/(n-k) · Σ u_t u_t' |
| HC2 | e_t / sqrt(1-h_t) | Σ u_t u_t' |
| HC3 | e_t / (1-h_t) | Σ u_t u_t' |
| HAC | e_t | Σ u_t u_t' + Σ_{j=1..L} (1 - j/(L+1)) Σ_t (u_t u_{t-j}' + u_{t-j} u_t') |

HAC with lag 0 runs exactly the HC0 arithmetic, so the two are equal bit for bit, not merely close. A leverage of 1 makes HC2/HC3 divide by zero; the result is NaN or Inf, which is what `sandwich` returns too.

## Distribution

Statistics are t with n-k degrees of freedom for every CovType, as `lmtest::coeftest` reports them and as `NeweyWestMeanTest` already does. statsmodels reads robust p-values from the normal distribution unless `use_t=True`; the docs say so.

## Lag conventions

`stats` uses its existing convention, now named: a negative lag is `stats.AutoLag` and picks `NeweyWestLag(n)`, 0 and above are used as given, and a lag not smaller than n is refused. The `quant` options are built for struct zero values, so 0 is automatic, a positive lag is used as given and a negative one is an error, for `FactorModelOptions.Lag` and `ForwardOptions.NeweyWestLag` alike. Turning the correction off is not a lag: `ForwardOptions.NoNeweyWest` does it (lag 0, variance divided by the number of months), and `FactorModel` points to `CovHC0`. `Lag` is ignored for the other CovTypes, the way `LogisticRegressionOptions.Ridge` is ignored without `SepRidge`, and the result records 0.

## Reference

The comparison runs statsmodels `OLS(...).fit(cov_type=..., use_t=True)`, with `cov_kwds={"maxlags": L, "use_correction": False}` for HAC, on seeded data with heteroskedastic AR(1) errors. R's `sandwich` is not installed on the machine this was written on; statsmodels implements the same formulas (HC0–HC3 from MacKinnon and White, HAC without the n/(n-k) factor), and the CI reference job already installs statsmodels.
