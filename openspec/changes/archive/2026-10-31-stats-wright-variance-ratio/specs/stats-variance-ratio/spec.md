## ADDED Requirements

### Requirement: Wright's rank and sign variance ratios match vrtest

`stats.WrightVarianceRatio` SHALL compute Wright's (2000) `R1` on average ranks, `R2` on their normal scores and `S1` on signs with a zero counted as a fall, equal to R's `vrtest::Wright` on the same series, and SHALL refuse no holding period, one outside 2 to n - 1, a negative `Replications`, more than one options value, and returns that are all equal.

#### Scenario: A series with ties and zeros

- **WHEN** the 80-value series of `testdata/wright_vrtest.R` is tested at q = 2, 5 and 10
- **THEN** R1, R2 and S1 equal vrtest's to within 1e-12 relative and Zeros is 6

### Requirement: Wright p-values come from a seeded simulation of the null

Each p-value of `WrightVarianceRatio` SHALL be `(1 + b) / (B + 1)` for `b` of `B` replications at least as far from 0 as the observed statistic, where a replication permutes the observed `R1` and `R2` scores and draws independent signs for `S1`; `B` SHALL default to 10,000 and the seed SHALL always apply, so the same input and seed give the same p-values.

#### Scenario: A trending series

- **WHEN** the 60 trending increments are tested with 999 replications
- **THEN** every p-value is 1/1000

### Requirement: Variance ratio results print as one table

`VarianceRatio` and `WrightVarianceRatio` SHALL return named slices of their per-period results whose `String` is the test's title followed by a header of the field names and one row per holding period, and whose `Show` prints it; the named slice SHALL be assignable to a plain slice of the results.

#### Scenario: Two holding periods

- **WHEN** VarianceRatio is called with q = 2 and 5 and the result is shown
- **THEN** it prints "Variance ratio", a header and two rows
