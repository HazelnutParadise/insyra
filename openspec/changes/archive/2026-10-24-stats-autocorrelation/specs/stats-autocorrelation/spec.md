## ADDED Requirements

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
