# Proposal: taifex-open-data

## Why

[#437](https://github.com/HazelnutParadise/insyra/issues/437): the institutional traders' futures and options positions, the large traders' open positions and the put/call ratio are standard inputs to Taiwanese market-sentiment and hedging studies, and `datafetch` had none of them.

## What Changes

- `datafetch.TAIFEX(cfg)` returns a `TAIFEXClient` on the exchange's keyless OpenAPI, with `InstitutionalFutures`, `InstitutionalOptions`, `LargeTradersFutures`, `LargeTradersOptions` and `PutCallRatio` (and `...Context` forms), each a typed `DataTable`.
- It reuses `TWStockConfig`, the limiter, retries and the throttling error, and reads either JSON or the CSV with Chinese headers that the OpenAPI returns for some tables regardless of the request.
- The OpenAPI holds the latest day only (20 days for the put/call ratio). Reading the website's CSV history is not done, and the documentation says so, which leaves that part of the issue open.
- Checked live: all five tables return rows (66, 30, 1,386, 320 and 20).

## Capabilities

### New Capabilities

- `taifex-open-data`: TAIFEX institutional positions, large traders and put/call ratio.

## Impact

`datafetch/taifex.go`, fixtures, tests, `Docs/datafetch.md`, both CHANGELOGs.
