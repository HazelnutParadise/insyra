# Proposal: ta-oscillators

## Why

[#481](https://github.com/HazelnutParadise/insyra/issues/481), batch 3 of 5: the oscillators.

## What Changes

- `ta.CCI`, `ta.Momentum`, `ta.ROC` (a percentage, as TA-Lib), `ta.UltimateOscillator`, `ta.StochRSI` (the same KDOptions as `KD`), `ta.ARBR`, `ta.CR`, `ta.ForceIndex` and `ta.ElderRay`.
- CR's midpoint is an option with four forms, because platforms disagree.

## Capabilities

### New Capabilities

- `ta-oscillators`: the oscillators above.

## Impact

`quant/ta/oscillators.go`, `quant/ta/indicators.go` (KD's smoothing is shared), tests, `Docs/quant-ta.md`, both CHANGELOGs.
