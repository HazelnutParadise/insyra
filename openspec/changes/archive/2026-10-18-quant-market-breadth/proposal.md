# Proposal: quant-market-breadth

## Why

[#448](https://github.com/HazelnutParadise/insyra/issues/448): `quant` had no market-breadth indicators, so a breadth study computed ADR, ADL, OBOS and the net advance ratio by hand.

## What Changes

- `quant.AdvanceDeclineRatio`, `AdvanceDeclineLine`, `OverboughtOversold` and `NetAdvanceRatio`, with the signatures the issue suggested.
- Missing days (nil or NaN) output NaN and are skipped; the window is the last N valid days. A zero denominator is NaN, never 0 or Inf.

## Capabilities

### New Capabilities

- `quant-market-breadth`: market-breadth indicators.

## Impact

`quant/breadth.go`, tests, `Docs/quant.md`, both CHANGELOGs.
