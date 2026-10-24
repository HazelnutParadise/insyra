# Proposal: taifex-history

## Why

[#437](https://github.com/HazelnutParadise/insyra/issues/437): the OpenAPI gives only the latest day (20 days for the put/call ratio), and the issue asked for the website's longer history as well.

## What Changes

- Five `...History(from, to)` methods on `TAIFEXClient`, same columns as the latest-day methods, oldest first, read from the website's CSV downloads (Big5), month by month.
- The website's limits (how far back each table goes, how long a range may be) were measured and are documented; a refused range is an error.

## Capabilities

### New Capabilities

- `taifex-history`: the history methods.

## Impact

`datafetch/taifex.go` (column keys for the website's headers, date format), `datafetch/taifex_history.go`, tests, `Docs/datafetch.md`, both CHANGELOGs.
