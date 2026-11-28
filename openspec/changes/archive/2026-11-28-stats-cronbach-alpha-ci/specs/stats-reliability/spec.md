## ADDED Requirements

### Requirement: CronbachAlpha reports Feldt's interval and Guttman's lambda 6

`CronbachAlpha` SHALL report Feldt's confidence interval for raw alpha at `ConfidenceLevel`, 0 meaning 0.95, and Guttman's lambda 6 from the correlation matrix, both agreeing with psych's `alpha.ci` and `G6(smc)`. A level outside (0, 1) SHALL be an error.

#### Scenario: Agreement with psych
- **WHEN** the scales of the alpha parity test are passed, and one at level 0.9
- **THEN** the interval and lambda 6 equal psych's within 1e-9, and the interval holds alpha

#### Scenario: A level outside (0, 1)
- **WHEN** `ConfidenceLevel` is 1, -0.5 or NaN
- **THEN** the call returns an error
