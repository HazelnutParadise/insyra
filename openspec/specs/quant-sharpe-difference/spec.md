# quant-sharpe-difference Specification

## Purpose
Testing whether two return series over the same periods have the same Sharpe ratio, with a closed formula cheap enough for thousands of series and an explicit result for a series that has no Sharpe ratio.

## Requirements

### Requirement: Two Sharpe ratios are compared with the Jobson-Korkie test corrected by Memmel
`quant.SharpeDifferenceTest` SHALL return the per-period Sharpe ratios, their difference, the Memmel-corrected statistic and its two-tailed and both one-tailed normal p-values, with a difference of 0 and a p-value of 1 for identical series.

#### Scenario: Size under the null
- **WHEN** 4000 pairs of correlated normal series with equal true Sharpe ratios are tested at 5%
- **THEN** between 3.7% and 6.4% of the tests reject

#### Scenario: Swapping the series
- **WHEN** the two series are swapped
- **THEN** the statistic changes sign and the two-tailed p-value is unchanged

### Requirement: A series without a Sharpe ratio is reported explicitly
`quant.SharpeDifferenceTest` SHALL return an error matching `ErrSharpeUndefined` when either series has zero standard deviation, and SHALL return an error for a nil list, a value that is not a finite number, series of different lengths, fewer than 3 observations and a non-finite risk-free rate.

#### Scenario: A strategy that never entered
- **WHEN** one series is all zeros
- **THEN** the error matches `ErrSharpeUndefined`
