# Proposal: quant-volume-indicators

## Why

[#456](https://github.com/HazelnutParadise/insyra/issues/456): testing Granville's on-balance volume on Taiwan and US data meant writing every OBV quantity by hand, since `quant` has no volume indicators.

## What Changes

- `quant.OBV`, `OBVFromReturns` (direction from a return series, for dividend-adjusted moves with raw volume), `SignedVolumeRatio` and `Divergence` (flags a new n-day price extreme the indicator does not confirm, with a cooldown).
- Inputs are refused the way the rest of `quant` refuses them, with errors naming the zero-based `row index`; the issue asked for 1-based rows, which the library no longer uses.
- `Divergence` replaces the issue's `NewHighDivergence`, because it flags lows as well as highs.
- `numericSeries` treats a typed nil list as nil.

## Capabilities

### New Capabilities

- `quant-volume-indicators`: volume indicators.

## Impact

`quant/volume.go`, `quant/bootstrap.go`, tests, `Docs/quant.md`, both CHANGELOGs.
