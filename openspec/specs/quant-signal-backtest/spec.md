# quant-signal-backtest Specification

## Purpose
Turning a delayed position signal from -1 (short) to 1 (long) into a net return series with cash returns and separate buy and sell costs, so a signal study measures what a trader could have earned.
## Requirements
### Requirement: A signal takes effect one row late
`quant.BacktestSignal` SHALL hold signal[t-1] during row t and InitialPosition during row 0, so a signal never earns the return of the row whose close produced it.

#### Scenario: A signal of 1, 1, 0, 0, 1
- **WHEN** the signal is 1, 1, 0, 0, 1
- **THEN** the positions are 0, 1, 1, 0, 0

### Requirement: Returns are net of cash and asymmetric costs
`quant.BacktestSignal` SHALL return, for each row, the position times the asset return plus the rest times the cash return, less BuyCost times the position added and SellCost times the position removed at the start of the row, and SHALL count the rows on which the position changed.

#### Scenario: Buying and selling
- **WHEN** the position goes from 0 to 1 with BuyCost 0.002 and from 1 to 0 with SellCost 0.001
- **THEN** the rows pay 0.002 and 0.001 respectively and Trades is 2

### Requirement: Unusable input is refused
`quant.BacktestSignal` SHALL return an error for a signal outside [-1, 1], lists of different lengths, a nil or non-finite series, a cost or borrow fee outside [0, 1) and an initial position outside [-1, 1].

#### Scenario: A position of 1.5
- **WHEN** a signal value is 1.5
- **THEN** an error names its row index

#### Scenario: A position of -1.5
- **WHEN** a signal value is -1.5
- **THEN** an error names its row index

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

