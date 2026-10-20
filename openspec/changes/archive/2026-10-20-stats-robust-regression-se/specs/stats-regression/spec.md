## ADDED Requirements

### Requirement: Ordinary least squares can report robust and HAC standard errors
`stats.LinearRegressionWithOptions(opts, y, xs...)` SHALL accept `LinearRegressionOptions{CovType, Lag}` and compute standard errors, t values, p values and confidence intervals from the covariance `opts.CovType` names: `CovOLS` (the classical one), `CovHC0`, `CovHC1`, `CovHC2`, `CovHC3` as R's `sandwich::vcovHC` defines them, or `CovHAC`, the Newey-West covariance with Bartlett weights 1 - j/(lag+1) and no small-sample factor. Coefficients, residuals and R² SHALL NOT depend on `CovType`.

#### Scenario: Matches statsmodels
- **WHEN** a seeded regression with heteroskedastic, autocorrelated errors is fitted with each CovType
- **THEN** standard errors, t values and p values agree with statsmodels `OLS.fit(cov_type=..., use_t=True)` within 1e-9, with `use_correction=False` and `maxlags` equal to the lag for HAC

#### Scenario: HAC with lag zero is HC0
- **WHEN** the same data are fitted with `CovHAC` and lag 0 and with `CovHC0`
- **THEN** the standard errors are bit-for-bit equal

### Requirement: The default regression is unchanged by the options
`stats.LinearRegression(y, xs...)` SHALL return the same numbers bit for bit as `LinearRegressionWithOptions(LinearRegressionOptions{}, y, xs...)` and as before the options existed. Inference for every CovType SHALL use the t distribution with n-k degrees of freedom.

#### Scenario: Zero options
- **WHEN** a regression is fitted with `LinearRegression` and with zero options
- **THEN** every field of the two results is identical and `CovType` is `CovOLS`

### Requirement: The covariance used is recorded and its settings validated
`LinearRegressionResult` SHALL record `CovType` and `Lag`. For `CovHAC` a negative `Lag` SHALL select `NeweyWestLag(n)` and the recorded `Lag` SHALL be the one used; for other CovTypes `Lag` SHALL be ignored and recorded as 0. The function SHALL return an error for an unknown CovType or a HAC lag not smaller than n. `CovType` SHALL print its constant name.

#### Scenario: Automatic HAC lag
- **WHEN** 120 observations are fitted with `CovHAC` and lag -1
- **THEN** the recorded lag is `NeweyWestLag(120)`

#### Scenario: Unknown CovType
- **WHEN** `CovType(99)` is passed
- **THEN** an error naming `CovType(99)` is returned
