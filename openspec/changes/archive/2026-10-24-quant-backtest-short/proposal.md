# Proposal: quant-backtest-short

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), item 4: a filter-rule study reports a long-only and a long-short version (Fama and Blume 1966), and `BacktestSignal` refuses a position below 0, so a short leg cannot be backtested. The owner chose to extend it on 2026-10-24 after a Fable review.

## What Changes

- `BacktestSignal` accepts a signal and an `InitialPosition` from -1 (fully short) to 1 (fully long). A call that succeeded before gives the same result; a -1 that used to be an error now runs.
- New `BacktestOptions.BorrowFee`: a per-row fee as a fraction of the short position, charged on every row held below 0 (default 0).
- The row return formula is unchanged. For a short (position -1) it is -assetReturn + 2·cashReturn: the capital and the sale proceeds held as collateral both earn the cash return. Costs already treat a position added as a buy and a position removed as a sell, so a flip from +1 to -1 pays 2·SellCost.
- Not in this change: a rebate rate below the cash rate as its own series, margin requirements, dividends modelled separately (use total returns).

## Capabilities

### Modified Capabilities

- `quant-signal-backtest`: positions from -1 to 1 and the borrow fee.

## Impact

`quant/signal_backtest.go`, `quant/signal_backtest_test.go`, `Docs/quant.md`, both CHANGELOGs. Additive for every call that succeeded before.
