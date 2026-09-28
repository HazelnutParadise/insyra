# Proposal: datafetch-exported-types

## Why

DF-3 of the API review ([#251](https://github.com/HazelnutParadise/insyra/issues/251)): every `datafetch` constructor returns a pointer to an unexported type. `TWStock` returns `*twStock`, `YFinance` returns `*yahooFinance`, its `Ticker` returns `*ticker`, `TWGeocoding` returns `*twGeocoder` and `GoogleMapsStores` returns `*googleMapsStoreCrawler`. A caller can hold the value in a local variable, but cannot name it in a struct field, a function parameter or a return type, so a program that builds one client at start-up and hands it to the code that uses it has to declare an interface of its own to hold it, as the CLI's `fetch tw` does.

## What Changes

- The five types are exported: `TWStockClient`, `YFinanceClient`, `YFTicker`, `TWGeocodingClient` and `GoogleMapsStoresClient`. Each constructor keeps its name and its parameters and returns the exported type, so every existing call compiles unchanged.
- Each client type is named after its constructor with `Client` appended, the same way `TWStockConfig` is named after it with `Config`, so a caller who knows the constructor can write the type. The per-symbol handle `Ticker` returns is `YFTicker`, following the `YF` prefix of `YFPeriod`, `YFHistoryParams` and `YFOptionChainTables`.
- Once a type is exported, a caller can declare its zero value, `var c datafetch.TWStockClient`, or call a method on a nil pointer, and neither went through the constructor. Today the TWSE/TPEx client, the geocoding client and the Google Maps client dereference a nil `*http.Client` or limiter in that case and panic. Every method of the five types now returns an error, or for the Google Maps client the usual `nil` with a warning, naming the constructor to use. The Yahoo Finance types already returned an error; their messages now name the exported types.

## Capabilities

### New Capabilities

- `datafetch-client-types`: each `datafetch` constructor returns an exported type, and a client that did not come from its constructor fails without panicking.

### Modified Capabilities

(none)

## Impact

- `datafetch/twstock.go`, `datafetch/yfinance.go`, `datafetch/geocoding.go`, `datafetch/googleMapsCommentCrawler.go` and the package's tests that name the old types. No behaviour of a client built by its constructor changes.
- `Docs/datafetch.md` (signatures and the zero-value note), both changelogs, `api-review.md` (DF-3), `delivery-status.md`.
- Callers outside the module are unaffected unless they relied on not being able to name the types.
