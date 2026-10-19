# Proposal: quant-technical-indicators

## Why

[#474](https://github.com/HazelnutParadise/insyra/issues/474): `quant` had RSI, OBV and the breadth indicators but not the most common technical indicators, and nothing to check a hand-built one against.

## What Changes

- `quant.SMA`, `EMA`, `MACD`, `Bollinger`, `TrueRange`, `ATR` and `KD`, with TA-Lib's definitions.
- `KD` has the recursive smoothing of Taiwanese brokers' software (the default) and TA-Lib's STOCH.
- Each is compared with TA-Lib on three seeded synthetic series.

## Capabilities

### New Capabilities

- `quant-technical-indicators`: moving averages, MACD, Bollinger bands, ATR and KD.

## Impact

`quant/indicators.go`, tests, `quant/testdata`, `Docs/quant.md`, both CHANGELOGs.
