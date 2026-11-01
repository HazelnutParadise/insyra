## ADDED Requirements

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
