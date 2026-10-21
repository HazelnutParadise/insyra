# Tasks

- [x] `CovType` with `String()`, `LinearRegressionOptions`, `LinearRegressionWithOptions`; `LinearRegression` delegates with zero options; result records `CovType` and `Lag` (`stats/regression.go`, `stats/enum_names.go`).
- [x] Sandwich covariance and variance-based inference helper in `stats/olsutil.go`; `computeCoeffInference` delegates without changing its arithmetic.
- [x] `FactorModelOptions` and the optional argument on `FactorModel`; result records `CovType` and `Lag`; fix the "OLS" field comments (`quant/factor.go`).
- [x] Tests: statsmodels reference for every CovType; HAC lag 0 equals HC0 bit for bit; zero options equal `LinearRegression` bit for bit; automatic lag; bad CovType and lag; enum names; `FactorModel` with and without options.
- [x] `Docs/stats.md`, `Docs/quant.md` (drop the "Newey–West is outside this API" sentence), entries in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] `openspec validate stats-robust-regression-se --strict` and `openspec validate --specs --strict` pass.
