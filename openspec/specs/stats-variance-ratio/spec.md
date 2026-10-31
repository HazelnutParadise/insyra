# stats-variance-ratio Specification

## Purpose
Testing the random-walk hypothesis with the Lo-MacKinlay variance ratio, alone or jointly over several holding periods with the Chow-Denning correction.
## Requirements
### Requirement: The variance ratio follows Lo and MacKinlay
`stats.VarianceRatio` SHALL return, for each holding period q, the ratio of the variance of overlapping q-period sums to the one-period variance using the unbiased estimators, the homoscedastic z, the heteroskedasticity-robust z*, and their two-tailed normal p-values.

#### Scenario: A random walk
- **WHEN** 1500 normal series of 400 returns are tested at q = 5
- **THEN** each statistic rejects between 3% and 7.5% of them at the 5% level

#### Scenario: Positively autocorrelated returns
- **WHEN** returns follow an AR(1) process with coefficient 0.4
- **THEN** VR(5) exceeds 1.2 and the robust p-value is below 0.001

### Requirement: Several holding periods are tested jointly
`stats.ChowDenning` SHALL return the largest absolute z over the holding periods, homoscedastic or robust as asked, and its p-value 1 - (2*Phi(max) - 1)^k.

#### Scenario: Three periods
- **WHEN** q is 2, 5 and 10
- **THEN** the p-value uses k = 3

### Requirement: Unusable input is refused
Both functions SHALL return an error for no holding period, a q below 2 or above n-1, a nil list, a value that is not a finite number and returns with no variance.

#### Scenario: A q of 1
- **WHEN** q is 1
- **THEN** an error names the allowed range

### Requirement: Wright's rank and sign variance ratios match vrtest

`stats.WrightVarianceRatio` SHALL compute Wright's (2000) `R1` on average ranks, `R2` on their normal scores and `S1` on signs with a zero counted as a fall, equal to R's `vrtest::Wright` on the same series, and SHALL refuse no holding period, one outside 2 to n - 1, a negative `Replications`, more than one options value, and returns that are all equal.

#### Scenario: A series with ties and zeros

- **WHEN** the 80-value series of `testdata/wright_vrtest.R` is tested at q = 2, 5 and 10
- **THEN** R1, R2 and S1 equal vrtest's to within 1e-12 relative and Zeros is 6

### Requirement: Wright p-values come from a seeded simulation of the null

Each p-value of `WrightVarianceRatio` SHALL be the equal-tailed two-sided `2 * min(1 + below, 1 + above) / (B + 1)`, capped at 1, for the `B` replications at or below and at or above the observed statistic, where a replication permutes the observed `R1` and `R2` scores and draws independent signs for `S1`; `B` SHALL default to 10,000 and the seed SHALL always apply, so the same input and seed give the same p-values.

#### Scenario: A trending series

- **WHEN** the 60 trending increments are tested with 999 replications
- **THEN** every p-value is 2/1000

### Requirement: Variance ratio results print as one table

`VarianceRatio` and `WrightVarianceRatio` SHALL return named slices of their per-period results whose `String` is the test's title followed by a header of the field names and one row per holding period, and whose `Show` prints it; the named slice SHALL be assignable to a plain slice of the results.

#### Scenario: Two holding periods

- **WHEN** VarianceRatio is called with q = 2 and 5 and the result is shown
- **THEN** it prints "Variance ratio", a header and two rows

