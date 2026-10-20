# quant-fama-macbeth Specification

## Purpose
Fama-MacBeth cross-sectional regressions on a long stock-by-date panel, testing whether each characteristic explains returns once the others are controlled for, with Newey-West standard errors on the time averages.

## Requirements
### Requirement: Every date is regressed on its own and the coefficients are averaged
`quant.FamaMacBeth` SHALL regress, for each date of a long table, the return on an intercept and the characteristics over the stocks whose return and characteristics are all present, and SHALL report the time average of the daily intercept and of each daily coefficient, the per-day table of date, intercept, coefficients, R² and stock count, and the mean of the finite daily R².

#### Scenario: An exact linear cross-section
- **WHEN** every day's return is 1 + 2·x1 − 3·x2 exactly
- **THEN** the intercept is 1, the coefficients are 2 and −3, and every daily R² is 1

### Requirement: The averages are tested with a Newey-West t-test
`quant.FamaMacBeth` SHALL test each average with `stats.NeweyWestMeanTest` on the series of daily estimates, choosing the lag automatically when NeweyWestLag is 0 and using the given lag when it is positive, and its standard errors SHALL equal `linearmodels.FamaMacBeth` with the Bartlett kernel, the same bandwidth and `debiased=False`.

#### Scenario: Agreement with linearmodels
- **WHEN** a seeded panel is fitted with lags 0, 3 and 5
- **THEN** coefficients, standard errors and t statistics match the linearmodels baseline to 1e-9 relative

### Requirement: The correction can be switched off, and a negative lag is refused
`quant.FamaMacBeth` SHALL, when NoNeweyWest is true, run the t-test with lag 0 (variance divided by T, equal to `linearmodels` with `cov_type="unadjusted"` and `debiased=False`) and report Lag 0, and SHALL return an error for a negative NeweyWestLag and for NoNeweyWest together with a non-zero NeweyWestLag.

#### Scenario: No correction
- **WHEN** NoNeweyWest is true
- **THEN** the standard errors equal `stats.NeweyWestMeanTest` with lag 0 on each daily series and Lag is 0

#### Scenario: Contradictory or negative lag
- **WHEN** NeweyWestLag is negative, or NoNeweyWest is true with NeweyWestLag 2
- **THEN** the call returns an error

### Requirement: A day that cannot be estimated is skipped, not fatal
`quant.FamaMacBeth` SHALL skip and list in Skipped a day with fewer usable stocks than MinObservations (at least k+2 for k characteristics) and a day whose cross-section is collinear, and SHALL return an error only when fewer than two days remain.

#### Scenario: A constant characteristic on one day
- **WHEN** one characteristic is the same for every stock on one date
- **THEN** that date is in Skipped with a collinearity reason and the other dates are estimated

### Requirement: Standardize uses per-day z-scores
`quant.FamaMacBeth` SHALL, when Standardize is set, replace each characteristic by its z-score over the day's used stocks with the sample standard deviation before regressing.

#### Scenario: Standardized slopes
- **WHEN** a day's characteristic is rescaled by 10 and Standardize is set
- **THEN** the day's coefficient is unchanged

### Requirement: Input that cannot be regressed is refused
`quant.FamaMacBeth` SHALL return an error for a nil table, no characteristic, a repeated characteristic, the return column listed as a characteristic, a column that is not found, a missing date, a text or infinite value, more than one options value, a negative lag, and a lag not smaller than the number of estimated days.

#### Scenario: A column that does not exist
- **WHEN** a characteristic is named "size" and the table has no such column
- **THEN** an error names the column

