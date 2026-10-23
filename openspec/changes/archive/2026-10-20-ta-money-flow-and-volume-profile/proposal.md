# Proposal: ta-money-flow-and-volume-profile

## Why

[#481](https://github.com/HazelnutParadise/insyra/issues/481), batch 4 of 5: the volume indicators.

## What Changes

- `ta.MFI`, `ta.AccumulationDistribution`, `ta.CMF`, `ta.ChaikinOscillator`, `ta.EaseOfMovement`, `ta.NVI`, `ta.PVI` and `ta.VolumeProfile` (point of control and value area per window).
- The ease-of-movement volume scale and the volume profile's bins and value area are options.

## Capabilities

### New Capabilities

- `ta-money-flow-and-volume-profile`: the volume indicators above.

## Impact

`quant/ta/money_flow.go`, `quant/ta/volume_profile.go`, tests, `Docs/quant-ta.md`, both CHANGELOGs.
