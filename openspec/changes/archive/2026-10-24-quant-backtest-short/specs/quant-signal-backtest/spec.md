## MODIFIED Requirements

### Requirement: Unusable input is refused
`quant.BacktestSignal` SHALL return an error for a signal outside [-1, 1], lists of different lengths, a nil or non-finite series, a cost or borrow fee outside [0, 1) and an initial position outside [-1, 1].

#### Scenario: A position of 1.5
- **WHEN** a signal value is 1.5
- **THEN** an error names its row index

#### Scenario: A position of -1.5
- **WHEN** a signal value is -1.5
- **THEN** an error names its row index

## ADDED Requirements

### Requirement: A negative position is a short

`quant.BacktestSignal` SHALL treat a position below 0 as a short of that share of the portfolio, earning the negative of the asset return and the cash return on the capital and on the proceeds, so a position of -1 returns -assetReturn + 2·cashReturn. It SHALL charge BorrowFee times the short position on every row held below 0.

#### Scenario: A short row

- **WHEN** the position is -1, the asset returns 0.10, the cash returns 0.001 and BorrowFee is 0.0004
- **THEN** the row returns -0.10 + 0.002 - 0.0004 before trading costs

### Requirement: A flip from long to short pays twice the sell cost

Moving from a position of 1 to -1 SHALL count as removing 2 of position and pay 2·SellCost, and Turnover for that row SHALL be 2.

#### Scenario: Flip

- **WHEN** the position goes from 1 to -1 with SellCost 0.01
- **THEN** the row pays 0.02 and Turnover is 2
