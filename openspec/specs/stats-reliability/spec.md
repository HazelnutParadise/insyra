# stats-reliability Specification

## Purpose
Gives the reliability of a scale of questionnaire items, so the internal consistency of a Likert scale can be checked in Insyra before its items are summed or factor-analysed.
## Requirements
### Requirement: CronbachAlpha computes a scale's reliability

`CronbachAlpha` SHALL return raw alpha, standardized alpha and the mean inter-item correlation of the items, and per item its mean, SD with n - 1, item-rest correlation and raw alpha with the item deleted, agreeing with R's `psych::alpha` on complete data. With two items, alpha with an item deleted SHALL be NaN.

#### Scenario: Agreement with psych
- **WHEN** a six-item scale of 120 rows, a two-item scale, a scale with a negatively correlated item, and one with a reverse-scored item are passed
- **THEN** every statistic equals psych 2.6.5's within 1e-9, apart from alpha if deleted on two items, which is NaN

### Requirement: CronbachAlpha removes incomplete rows listwise

A row with a NaN or ±Inf in any item SHALL be left out and counted in `RowsDropped`; a text cell SHALL be an error naming its row and column from 0.

#### Scenario: Two rows with a missing answer
- **WHEN** two rows with a NaN or an infinity are added to a complete table
- **THEN** alpha is unchanged, `N` is the complete rows and `RowsDropped` is 2

### Requirement: CronbachAlpha takes items and reverse-scored items as column selectors

`items` and `ReverseScored` SHALL be column selectors of the table passed in, `nil` items meaning every column. A reverse-scored item SHALL be negated for the statistics while its `Mean` stays its own. A repeated, unknown or out-of-scale selector, a constant item, items whose total never varies, a non-finite result, fewer than two items or rows, and more than one options value SHALL be errors.

#### Scenario: The same items three ways
- **WHEN** the same three items are given by name, by letter and by position
- **THEN** the three results have the same alpha

### Requirement: CronbachAlpha reports Feldt's interval and Guttman's lambda 6

`CronbachAlpha` SHALL report Feldt's confidence interval for raw alpha at `ConfidenceLevel`, 0 meaning 0.95, and Guttman's lambda 6 from the correlation matrix, both agreeing with psych's `alpha.ci` and `G6(smc)`. A level outside (0, 1) SHALL be an error.

#### Scenario: Agreement with psych
- **WHEN** the scales of the alpha parity test are passed, and one at level 0.9
- **THEN** the interval and lambda 6 equal psych's within 1e-9, and the interval holds alpha

#### Scenario: A level outside (0, 1)
- **WHEN** `ConfidenceLevel` is 1, -0.5 or NaN
- **THEN** the call returns an error

