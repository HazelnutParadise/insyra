# Proposal: quant-rsi-and-sharpe-difference

## Why

[#464](https://github.com/HazelnutParadise/insyra/issues/464): verifying RSI over thousands of stocks needed an RSI and a test of whether a strategy's Sharpe beats buy-and-hold, and `quant` had neither.

## What Changes

- `quant.RSI(prices, n, RSIOptions)` with Wilder or simple averaging and a choice for missing prices (error, carry the price over, or restart the series).
- `quant.SharpeDifferenceTest(a, b, riskFree)`, the Jobson-Korkie statistic with Memmel's correction, with two-tailed and both one-tailed p-values. A series with no variance returns `ErrSharpeUndefined`, an explicit result instead of NaN.
- Not done from the issue: the Ledoit-Wolf HAC or bootstrap version, Storey's pi0, and the KD, Williams %R and MACD oscillators. Each is its own change; the issue marks the oscillators as later.
- Evidence: Wilder RSI(14) on the StockCharts example reproduces its published values; the simple variant and the Sharpe ratios were checked against an independent Python calculation; the test statistic's variance formula is checked by a simulation showing a 5% test rejects 5% under equal Sharpe ratios.

## Capabilities

### New Capabilities

- `quant-rsi`: the relative strength index.
- `quant-sharpe-difference`: the Sharpe difference test.

## Impact

`quant/rsi.go`, `quant/sharpe_difference.go`, tests, `Docs/quant.md`, both CHANGELOGs.
