# quant-signal-backtest Specification

## Purpose
Turning a delayed 0-to-1 position signal into a net return series with cash returns and separate buy and sell costs, so a signal study measures what a trader could have earned.

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
`quant.BacktestSignal` SHALL return an error for a signal outside [0, 1], lists of different lengths, a nil or non-finite series, a cost outside [0, 1) and an initial position outside [0, 1].

#### Scenario: A position of 1.5
- **WHEN** a signal value is 1.5
- **THEN** an error names its row index
