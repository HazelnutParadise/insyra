# Proposal: datafetch-fugle-technical

## Why

[#440](https://github.com/HazelnutParadise/insyra/issues/440), stage 4b: the second of four changes for the remaining stock endpoints, approved by the owner on 2026-10-30 ("do all the Fugle stock endpoints"). Fugle serves five technical indicators computed from its own candles.

## What Changes

- `client.Stock.TechnicalSMA`, `TechnicalRSI`, `TechnicalKDJ`, `TechnicalMACD` and `TechnicalBB`, each with a Context form, return a table with `date` and the indicator's columns.
- The timeframe and every period are required positional parameters in the order of Fugle's query parameters, because Fugle requires them and has no default.
- `FugleTechnicalOptions{From, To}` is the one optional trailing value: both or neither, and refused with a minute timeframe, which Fugle always answers with the last 30 days.
