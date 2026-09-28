# Proposal: datafetch-context

## Why

DF-2 of the API review ([#250](https://github.com/HazelnutParadise/insyra/issues/250)): no `datafetch` method takes a `context.Context`, and the request limiter waits under `context.Background()`. A ten-year `DailyPrices` backfill is about 120 requests spaced by `Interval`, a `ReverseCols` batch can wait an hour for the geocoding quota, and `GetReviews` sleeps up to five seconds between pages; none of them can be stopped once started. A server that fetches on behalf of an HTTP request cannot pass the request's context, so the fetch runs on after the client has gone.

The root package and `py` already solve this one way: `ReadSQLContext`, `ToSQLContext` and `RunCodeContext` take the context, and the plain name calls them with `context.Background()`.

## What Changes

- Every method that fetches gets a `...Context` form taking `ctx context.Context` first, and the plain method becomes a call to it with `context.Background()`:
  - `TWStockClient`: `DailyPricesContext`, `DailyPricesAdjustedContext`, `ExRightsContext`, `InstitutionalTradesContext`, `MarginBalanceContext`, `AllDailyQuotesContext`.
  - `TWGeocodingClient`: `ReverseContext`, `ReverseColsContext`, `ReverseTableContext`.
  - `YFTicker`: a `...Context` form for each of the 26 methods that request data, such as `HistoryContext`, `QuoteContext` and `OptionChainContext`. `Earnings`, `Sustainability`, `FundsData` and `TopHoldings` return "not supported" without a request and get none.
  - `GoogleMapsStoresClient`: `SearchContext`, `GetReviewsContext`.
- The context reaches every wait and every request. The limiter waits under it, the retry backoff stops when it is done, and each HTTP request is built with it, so cancelling ends a request already sent. A call whose context is done returns `ctx.Err()`, so `errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)` tell a cancellation from a failure. A cancelled request is not retried, and `ReverseContext` reports a request its context cut off as the context's error, not as `ErrGeocodeTimeout`, which keeps meaning that the service did not answer within `Timeout`.
- `ReverseColsContext` stopped part-way returns the rows it resolved, with the rest marked `pending` and `ctx.Err()`, the way it already handles an exhausted quota, so a batch can be resumed.
- `GetReviewsContext` and `SearchContext` report a cancellation as they report any failure, a `nil` result and a warning.
- go-yfinance has no way to stop a call once it has started, and one call can make several requests. A Yahoo Finance method whose context ends while its call is running returns `ctx.Err()` at once and leaves the call to run to its end in the background, where it may still send its remaining requests; its result is discarded.
- A `nil` context is refused with an error instead of panicking.

## Capabilities

### New Capabilities

- `datafetch-cancellation`: how a `context.Context` reaches every `datafetch` fetch, and what a cancelled call returns.

### Modified Capabilities

(none)

## Impact

- `datafetch/twstock.go`, `datafetch/geocoding.go`, `datafetch/googleMapsCommentCrawler.go`, `datafetch/yfinance.go` and their tests. The Yahoo Finance methods are rewritten around two shared helpers, one that runs a go-yfinance call under a context and one that turns its result into a named table; what each method fetches and how its table is built does not change.
- The plain methods behave as before. `datafetch/internal/limiter` is not changed.
- `Docs/datafetch.md`, `skills/insyra/SKILL.md` (the `...Context` convention now covers the network fetchers), both changelogs, `api-review.md` (DF-2), `delivery-status.md`, and an `AGENTS.md` follow-up for a defect found on the way: only `History` and `Quote` honour `YFinanceConfig.Interval` and `Retries`.
