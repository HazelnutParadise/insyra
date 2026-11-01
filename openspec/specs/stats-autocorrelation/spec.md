# stats-autocorrelation Specification

## Purpose
The autocorrelation of a series by lag, with a white-noise test and a heteroskedasticity-robust test, for checking whether a return series is predictable from its own past.
## Requirements
### Requirement: Autocorrelation estimates the autocorrelation by lag

`Autocorrelation(series, maxLag)` SHALL return one result for each lag from 1 to `maxLag`, whose `Rho` is the sum of lagged products of deviations from the mean divided by the sum of squared deviations, the value R's `acf` gives.

#### Scenario: Agreement with R

- **WHEN** `Autocorrelation` is called on a series for which R's `acf` gives the first four autocorrelations
- **THEN** each `Rho` equals R's value to within 1e-12

### Requirement: Autocorrelation tests each lag against white noise and robustly

Each result SHALL carry `T`, equal to the square root of the number of observations times `Rho`, with its two-tailed normal p-value `P`, and `TRobust`, equal to `T` divided by the square root of Lo and MacKinlay's (1988) variance delta, with its two-tailed normal p-value `PRobust`.

#### Scenario: The robust statistic follows its definition

- **WHEN** delta is computed from the sum of squared deviations times squared lagged deviations
- **THEN** `TRobust` equals the square root of n times `Rho` over the square root of delta

### Requirement: Autocorrelation refuses input it cannot test

`Autocorrelation` SHALL return an error for a nil or empty list, a value that is not a finite number, a constant series, a `maxLag` below 1 or above the number of observations minus 1, and a lag at which the robust variance is zero.

#### Scenario: A sparse series

- **WHEN** `Autocorrelation` is called on `[-1, 0, 1]` with `maxLag` 1
- **THEN** it returns an error naming the lag, not a result holding NaN

### Requirement: Autocorrelation results print as one table

`Autocorrelation` SHALL return `AutocorrelationResults`, a named slice of its per-lag results whose `String` is "Autocorrelation" followed by a header of the field names and one row per lag, and whose `Show` prints it; it SHALL be assignable to a plain `[]AutocorrelationResult`.

#### Scenario: Three lags

- **WHEN** Autocorrelation is called with maxLag 3 and the result is shown
- **THEN** it prints "Autocorrelation", a header and three rows

### Requirement: Autocorrelation reports the Ljung-Box statistic over the first lags

The result for lag `m` SHALL carry `Q`, equal to `n(n+2)` times the sum over lags 1 to `m` of `Rho_j^2/(n-j)`, and `PQ`, its upper-tail chi-squared probability on `m` degrees of freedom, with no adjustment for fitted parameters.

#### Scenario: Agreement with R and statsmodels

- **WHEN** `Autocorrelation` is called on a series for which R's `Box.test(type = "Ljung-Box", fitdf = 0)` and statsmodels' `acorr_ljungbox` give the statistic and p-value at each lag
- **THEN** `Q` equals both within a relative 1e-10 and `PQ` equals both within 1e-12

### Requirement: Autocorrelation reports a robust portmanteau statistic

The result for lag `m` SHALL carry `QRobust`, equal to the sum of `TRobust_j^2` over lags 1 to `m` (the robust Box-Pierce statistic of Lobato, Nankervis and Savin 2001), and `PQRobust`, its upper-tail chi-squared probability on `m` degrees of freedom.

#### Scenario: The robust statistic is the sum of the squared robust t

- **WHEN** `Autocorrelation` is called with `maxLag` 10
- **THEN** each `QRobust` equals the running sum of `TRobust^2` and matches the value computed from the definition in R

#### Scenario: The table shows the new columns

- **WHEN** the result is shown
- **THEN** the header lists `Q`, `PQ`, `QRobust` and `PQRobust` after `PRobust`

