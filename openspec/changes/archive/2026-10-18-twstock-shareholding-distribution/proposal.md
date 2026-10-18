# Proposal: twstock-shareholding-distribution

## Why

[#436](https://github.com/HazelnutParadise/insyra/issues/436): the TDCC's weekly shareholding distribution, the usual view of big holders against small ones, was not available in `datafetch`.

## What Changes

- `TWStockClient.ShareholdingDistribution()` and its `Context` form read the TDCC open-data CSV into a table with a `time.Time` date, numeric holders, shares and percent, and a readable label for each of the 17 levels.
- The table holds the latest week only; the TDCC publishes no history there, which the documentation says.
- The request goes through the client's limiter, retries and throttling error. To share them, `doJSON` now sits on a new `doGet`, and the status and size handling of `readJSONResponse` moved to `readBodyResponse`; JSON behaviour is unchanged and the existing tests pass untouched.
- Checked against the live file for 2026-10-16 (4,103 securities, 17 levels each).

## Capabilities

### New Capabilities

- `twstock-shareholding-distribution`: the TDCC weekly shareholding distribution.

## Impact

`datafetch/twstock_tdcc.go`, `datafetch/twstock.go`, fixtures, tests, `Docs/datafetch.md`, both CHANGELOGs.
