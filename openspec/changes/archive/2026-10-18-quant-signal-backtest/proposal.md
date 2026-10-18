# Proposal: quant-signal-backtest

## Why

[#442](https://github.com/HazelnutParadise/insyra/issues/442): turning a position signal into a net return series, with the signal acting one row late, cash earning a return and buying and selling costing differently, was hand-written in Python.

## What Changes

- `quant.BacktestSignal(signal, assetReturns, opts)` returns the net returns, effective positions, turnover and number of trades.
- The position during row t is signal[t-1]; row 0 is held at InitialPosition. Buy and sell costs are fractions of the position added and removed.

## Capabilities

### New Capabilities

- `quant-signal-backtest`: a delayed position signal into net returns.

## Impact

`quant/signal_backtest.go`, tests, `Docs/quant.md`, both CHANGELOGs.
