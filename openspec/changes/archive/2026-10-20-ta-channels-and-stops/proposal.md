# Proposal: ta-channels-and-stops

## Why

[#481](https://github.com/HazelnutParadise/insyra/issues/481) asked for the other common indicators on top of #474. This is the first of five batches: channels and stops.

## What Changes

- `ta.Keltner` (Raschke), `ta.KeltnerClassic` (1960), `ta.Donchian`, `ta.SuperTrend`, `ta.ParabolicSAR` and `ta.Chandelier`, each returning named lines.
- Parabolic SAR is compared with TA-Lib; the others with implementations written from their definitions (`quant/ta/testdata/channels_baseline.py`).
- `Err()` is not used: these return an error, as the rest of `ta` does.

## Capabilities

### New Capabilities

- `ta-channels-and-stops`: channel and trailing-stop indicators.

## Impact

`quant/ta/channels.go`, tests, `Docs/quant-ta.md`, both CHANGELOGs.
