# Proposal: quant-divergence-skips-nan

## Why

`quant.RSI` and `quant.SignedVolumeRatio` return NaN for their first rows, but `quant.Divergence` refused any NaN, so `Divergence(close, RSI(close, 14), ...)`, the natural use, failed with "indicator contains a non-finite value at row index 0". Found in a review of the new API.

## What Changes

- `Divergence` treats a day as 0 when the indicator on that day or on any of the previous n days is NaN. A NaN price, a non-numeric cell and an infinite indicator value are still errors.

## Impact

`quant/volume.go`, `quant/bootstrap.go` (a NaN-tolerant reader beside `numericSeries`), tests, `Docs/quant.md`, both CHANGELOGs.
