## ADDED Requirements

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
