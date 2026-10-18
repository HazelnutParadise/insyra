# Proposal: quant-zigzag-regimes

## Why

[#442](https://github.com/HazelnutParadise/insyra/issues/442): testing Dow theory needed turning points and bull and bear stretches, and `quant` had neither, so the front half of the study was written in Python.

## What Changes

- `quant.ZigZag(prices, threshold)` returns the turning points with the row of the extreme and the row that confirmed it. It is causal, so a cut-short series gives exactly the pivots of the full series confirmed before the cut.
- `quant.Regimes(pivots, opts)` turns pivots into bull and bear stretches, with `MinPhase` and `MinCycle` as in Pagan and Sossounov (2003).
- Not done from the issue: the paired bootstrap test of a Sharpe difference, which stays open.

## Capabilities

### New Capabilities

- `quant-zigzag-regimes`: turning points and market regimes.

## Impact

`quant/zigzag.go`, tests, `Docs/quant.md`, both CHANGELOGs.
