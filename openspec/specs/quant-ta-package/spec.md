# quant-ta-package Specification

## Purpose
Keeping the technical-analysis tools in their own package, apart from the tools that evaluate trading strategies and hypotheses.

## Requirements

### Requirement: The technical-analysis tools live in quant/ta
The package `quant/ta` SHALL hold the technical-analysis tools (moving averages, MACD, Bollinger bands, ATR, KD, RSI, the volume and market-breadth indicators, the bias ratio, turning points and regimes), `quant` SHALL NOT export them, and the package documentation and `Docs/quant-ta.md` SHALL say what the October 2026 study found about each.

#### Scenario: Importing an indicator
- **WHEN** a program needs the relative strength index
- **THEN** it imports `github.com/HazelnutParadise/insyra/quant/ta` and calls `ta.RSI`

### Requirement: Tools that evaluate a hypothesis stay in quant
`quant` SHALL keep the performance and risk metrics, event studies, signal buckets, cross-sectional sorting, signal backtests, the Sharpe difference test and text tone, which do not depend on `quant/ta`.

#### Scenario: Evaluating a signal
- **WHEN** a program tests a signal built with `ta.Bias` using `quant.SignalBuckets`
- **THEN** both packages are imported and `quant/ta` does not import `quant`
