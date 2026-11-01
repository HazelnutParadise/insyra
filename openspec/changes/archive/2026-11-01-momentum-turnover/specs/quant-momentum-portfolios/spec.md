## ADDED Requirements

### Requirement: MomentumPortfolios reports each group's turnover

`MomentumResult.Turnover` SHALL have the rows and columns of `Groups`, each cell the sum over stocks of the absolute difference between the stock's weight in the month and its weight of the month before moved by that month's return and renormalised, the weight being the sum over the overlapping portfolios of `1/Hold` divided by the number of that portfolio's group stocks with a return in the month.

#### Scenario: One-month hold

- **WHEN** a group holds A and D at 0.5 each, they return 0.05 and -0.05, and the next month it holds A and B at 0.5 each
- **THEN** the group's turnover that month is |0.5 - 0.525| + 0.5 + 0.475 = 1

#### Scenario: Overlapping portfolios add up per stock

- **WHEN** Hold is 2 and a stock is in the group of both overlapping portfolios
- **THEN** its weight is the sum of its two shares before the difference is taken

### Requirement: MomentumPortfolios leaves turnover undefined where the holdings are unknown

A turnover cell SHALL be NaN in the first reported month, in a month after a skipped one, when the group return is NaN in the month or the month before, and when the moved weights sum to zero or less.

#### Scenario: After a skipped month

- **WHEN** a month is skipped because its portfolio could not be formed
- **THEN** the next reported month's turnover is NaN in every group

### Requirement: MomentumPortfolios reports the spread's turnover

`MomentumResult.LongShortTurnover` SHALL hold, for each row of `Turnover`, the turnover of the last group plus that of the first.

#### Scenario: Winners and losers both trade

- **WHEN** the winners' turnover is 1 and the losers' is 22/21 in a month
- **THEN** the spread's turnover that month is 1 + 22/21
