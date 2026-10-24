# quant-momentum-portfolios Specification

## Purpose
Jegadeesh and Titman momentum portfolios with overlapping holding periods, so a study of whether past winners keep winning does not hand-write the ranking and averaging.
## Requirements
### Requirement: MomentumPortfolios ranks on past compounded returns

`quant.MomentumPortfolios` SHALL rank, each month, the stocks whose returns are all present over the formation months ending Skip months before the month, by the compounded return (the product of 1 plus each return, less 1), and cut them into groups with QuantileSort's rank-tie rule, group 1 holding the lowest.

#### Scenario: One-month formation

- **WHEN** four stocks returned 0.10, 0.05, -0.05 and -0.10 in a month and two groups are asked for
- **THEN** the two highest form the winners and the two lowest the losers

### Requirement: MomentumPortfolios averages overlapping portfolios

Each group SHALL be held for Hold months (1 when Hold is 0) with equal weights, a stock with no return in a holding month left out of that month's average, and a month's return for a group SHALL be the equal-weighted average over the Hold portfolios formed in the Hold months before it.

#### Scenario: Two overlapping portfolios

- **WHEN** Hold is 2
- **THEN** a month's winners return is the mean of the winners formed one month earlier and the winners formed two months earlier

### Requirement: MomentumPortfolios reports complete months and what it skipped

A month SHALL be reported only after Hold portfolios have been formed, and LongShort SHALL be the last group's return minus the first group's for each reported month. A month that averages a portfolio that could not be formed, for fewer ranked stocks than groups, SHALL be left out and listed in Skipped with the reason.

#### Scenario: Too few ranked stocks

- **WHEN** a formation month has returns for two stocks and three groups are asked for
- **THEN** the portfolios it would form are skipped and the months that average them are listed in Skipped

### Requirement: MomentumPortfolios refuses unusable input

`quant.MomentumPortfolios` SHALL return an error for formation below 1, groups below 2, a negative Skip or Hold, more than one options value, a nil table, a missing column or date or stock, a non-numeric or infinite return, a stock appearing twice in a month, and fewer months than the formation, Skip and Hold need.

#### Scenario: A stock twice in a month

- **WHEN** one stock has two rows in the same month
- **THEN** an error names the row index

