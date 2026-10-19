## ADDED Requirements

### Requirement: The mean of an autocorrelated series is tested with a Newey-West standard error
`stats.NeweyWestMeanTest` SHALL return the mean, the Newey-West standard error with Bartlett weights 1 - j/(lag+1), the t statistic, the two-tailed p-value from the t distribution with n-1 degrees of freedom, the lag used and n, with no small-sample correction, and SHALL use floor(4 * (n/100)^(2/9)) as the lag when given a negative lag.

#### Scenario: Lag zero is the n-divisor t-test
- **WHEN** the lag is 0
- **THEN** the statistic equals the one-sample t statistic times sqrt(n/(n-1))

#### Scenario: Automatic lag
- **WHEN** the lag is negative and n is 20
- **THEN** the lag used is 2

### Requirement: A series the test cannot use is refused
`stats.NeweyWestMeanTest` SHALL return an error for fewer than two observations, a nil list, a value that is not a finite number, or a lag not smaller than n.

#### Scenario: Lag equal to n
- **WHEN** three observations are given with lag 3
- **THEN** an error is returned
