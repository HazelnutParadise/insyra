## ADDED Requirements

### Requirement: Divergence accepts an indicator that starts with NaN
`quant.Divergence` SHALL treat a day as 0 when the indicator on that day or on any of the previous n days is NaN, and SHALL still return an error for a NaN price, a non-numeric cell or an infinite indicator value.

#### Scenario: RSI as the indicator
- **WHEN** the indicator is the output of `quant.RSI`, whose first rows are NaN
- **THEN** no error is returned and the days whose window reaches a NaN are 0
