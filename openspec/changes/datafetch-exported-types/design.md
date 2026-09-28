# Design: datafetch-exported-types

## Names

A type cannot share its name with the constructor that returns it, so `TWStock` cannot return a type called `TWStock`. The package already pairs each constructor with a config named after it (`TWStockConfig`, `YFinanceConfig`, `TWGeocodingConfig`); appending `Client` the same way gives one rule for all four constructors, including `GoogleMapsStores`, whose options type already starts with `GoogleMapsStore`. The review suggested an interface as the alternative. An interface would be a second thing to keep in step with the methods, and a caller who wants one for a test fake can still declare it, as the CLI does.

`Ticker` returns a handle bound to one symbol rather than a client, so it is named after what it is, with the `YF` prefix every Yahoo Finance type in the package carries.

## The zero value

Exporting a type lets a caller build it without the constructor. The constructors do work a zero value cannot redo on demand: they validate and default the config, create the HTTP client with its timeout and the limiter, and for Yahoo Finance create the underlying go-yfinance client and register the finalizer that closes it. Making the zero value usable would mean repeating that lazily behind a `sync.Once` in every method, for a way of building the client nobody needs. Each method instead checks that its receiver came from the constructor, which is one comparison, and says which constructor to call. This matches the rule that the library never panics: without the check, a zero-value TWSE/TPEx client dereferences a nil `*http.Client`.

The Google Maps client reports every failure as a `nil` result with a warning, so a client not built by `GoogleMapsStores` is reported the same way rather than introducing an error return.
