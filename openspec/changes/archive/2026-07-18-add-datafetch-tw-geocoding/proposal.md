## Why

Insyra's `datafetch` package can pull financial data and Google Maps reviews, but has no way to turn geographic coordinates into administrative regions — a common need when analysing location-tagged datasets (survey responses, sensor logs, store lists). The `geocoding.zuola.com` reverse-geocoding API maps a `(lat, lng)` point to a Taiwan county/town/village at no cost, making it a natural fit for a Go data-analysis library that already targets Taiwanese use cases.

## What Changes

- Add a `TWGeocoding` reverse-geocoding fetcher to `datafetch`, following the existing `YFinance` config-driven, stateful pattern.
  - Single lookup: `Reverse(lat, lng) (*ReverseGeocodeResult, error)` returning a typed struct with the full county/town/village fields.
  - Batch lookups over Insyra data structures: `ReverseCols(lat, lng *insyra.DataList)`, `ReverseTable(dt, latCol, lngCol)` (Excel-index columns) and `ReverseTableByColName(dt, latColName, lngColName)` (named columns), each returning a `*insyra.DataTable` enriched with region columns plus a per-row `GeocodeStatus` column.
- Handle the free tier's **15 requests/hour** limit as a first-class concern: parse `X-Ratelimit-*` response headers into a `RateLimitError{Limit, Remaining, ResetAt}`, dedup identical coordinates within a batch, tolerate per-row failures, and stop cleanly on HTTP 429 while returning all rows already resolved.
- Provide an optional, opt-in `GeocodeCache` (in-memory and file-backed built-ins) so repeated or cross-run lookups avoid spending the scarce request budget.
- Expose stable sentinel errors: `ErrGeocodeNotFound` (point outside any TW village), `ErrGeocodeTimeout`, `ErrGeocodeRateLimited`.
- Keep docs and agent skills in sync in this same change: a new Geocoding section in `Docs/datafetch.md` and updated `skills/insyra/` references.

## Capabilities

### New Capabilities
- `datafetch-tw-geocoding`: Reverse-geocode Taiwan coordinates to administrative regions via the datafetch package, including single and batch lookups over Insyra data structures, rate-limit-aware error handling, and optional result caching.

### Modified Capabilities
<!-- None. datafetch has no existing capability spec; this is purely additive. -->

## Impact

- **New code**: `datafetch/geocoding.go`, `datafetch/geocoding_errors.go`, `datafetch/geocoding_test.go`.
- **Docs**: new Geocoding section in `Docs/datafetch.md`; `skills/insyra/` reference updates. `README.md`/`README_TW.md` package tables unchanged (`datafetch` is an existing package, no new row).
- **Dependencies**: none added — uses the standard library `net/http` plus the already-vendored `github.com/goccy/go-json`.
- **External service**: depends on the third-party `geocoding.zuola.com` endpoint; availability and the 15 req/hour free quota are outside our control. Endpoint is overridable via `BaseURL` (for tests/mocks and a future paid or self-hosted tier).
- **Testing**: unit tests run against an `httptest.Server` mock (no live calls by default) to avoid consuming the quota; any live test sits behind a build tag.
