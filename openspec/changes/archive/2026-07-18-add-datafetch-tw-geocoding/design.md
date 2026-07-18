## Context

`datafetch` already hosts two fetchers with two different styles: the older `GoogleMapsStores()` (returns `nil` on failure, no error, vendor-named) and the newer `YFinance(cfg)` (config-driven, stateful, `(result, error)` returns, an internal `limiter.IntervalLimiter`, sentinel errors in a dedicated `*_errors.go`, JSON via `github.com/goccy/go-json`). The newer YFinance shape is the one to follow.

The target service is `geocoding.zuola.com`. Verified against the live API:
- Single endpoint, reverse-only: `GET /api/reverse?lat=..&lng=..` → Taiwan county/town/village. No forward geocoding, no auth.
- Success (200): `{"ok":true,"query":{...},"result":{"villcode","county_name","town_name","village_name","village_eng","county_id","county_code","town_id","town_code"}}`.
- Not found (404): `{"ok":false,"error":"not_found","message":"No village polygon contains this point."}`.
- Free quota **15 requests/hour per IP**, surfaced via `X-Ratelimit-Limit / -Remaining / -Reset` headers (reset is a unix timestamp on the hour); exhaustion is expected to return HTTP 429.
- `Cache-Control: max-age=86400`; village boundaries are effectively static, so results are highly cacheable.

The 15 req/hour quota is the dominant design constraint and shapes the batch, cache, and retry decisions below.

## Goals / Non-Goals

**Goals:**
- A `TWGeocoding` fetcher mirroring the YFinance config/stateful/error conventions.
- Ergonomic single lookup returning a typed struct, plus batch lookups that enrich an `insyra.DataTable`.
- Make the quota survivable: in-batch dedup, per-row fault tolerance, precise rate-limit signalling, optional caching.
- Tests that never touch the live API by default (quota-safe).

**Non-Goals:**
- Forward geocoding (address → coordinate). The service does not offer it.
- A provider-agnostic geocoding abstraction / multi-provider plugin layer. Premature for a single reverse-only endpoint; `BaseURL` already covers a future paid or self-hosted tier without an interface.
- Spatial/proximity caching (rounding nearby points to one key). Correctness risk near village boundaries; out of scope.

## Decisions

### D1 — Follow the YFinance config/stateful pattern; name it `TWGeocoding`
`TWGeocoding(cfg TWGeocodingConfig) (*twGeocoder, error)`. Honest scope: Taiwan-specific, reverse-only today, room to add `Forward*` later without renaming.
- **Alternatives:** a bare `ReverseGeocode(lat,lng)` top-level function (rejected: the generic "Geocoding" name over-promises forward support and has nowhere to hold limiter/cache state); `ZuolaGeocoding` vendor-name (rejected: obscure vendor, binds the public API to one provider).

### D2 — Config fields mirror `YFinanceConfig`
`Timeout` (default 15s), `Interval` (client-side throttle, default 0), `UserAgent`, `Retries`, `RetryBackoff`, `BaseURL` (default official endpoint), `Cache GeocodeCache` (nil = disabled). Reuse `datafetch/internal/limiter.IntervalLimiter` for `Interval`. `normalize()` validates and applies defaults, matching YFinance.

### D3 — Two return shapes: typed struct for single, DataTable for batch
`Reverse(lat,lng) (*ReverseGeocodeResult, error)` returns a struct with all nine fields (best for single use). Batch methods return `*insyra.DataTable` (best for analysis). `ReverseGeocodeResult` also gets `ToDataTable()` for convenience.

### D4 — Batch column addressing mirrors insyra's own split
insyra distinguishes index vs. name with separate methods (`GetCol("A")` vs `GetColByName("lat")`) precisely because a bare string is ambiguous. Batch API follows suit:
- `ReverseTable(dt, latCol, lngCol string)` — Excel-style index, consistent with `GetCol`/`UpdateCol`/`FilterColsByColIndex*`.
- `ReverseTableByColName(dt, latColName, lngColName string)` — name, consistent with `GetColByName`/`FilterColsByColNameEqualTo`.
- `ReverseCols(lat, lng *insyra.DataList)` — the unambiguous primitive both wrappers delegate to.
- **Alternative:** a single `string` param (rejected: reintroduces the exact index/name ambiguity insyra deliberately avoids).

### D5 — Result columns + a `GeocodeStatus` column
Batch output keeps the input coordinates and appends `County, Town, Village, VillageEng, VillCode, CountyCode, TownCode, CountyID, TownID` plus `GeocodeStatus` (`ok` / `not_found` / `pending` / an error string). Failed rows keep empty region cells so the table stays rectangular and downstream code can filter on status.

### D6 — Batch fault tolerance and quota discipline
- **Dedup:** collapse identical `(lat,lng)` pairs and issue at most one request each — safe and the biggest quota win.
- **Per-row tolerance:** `not_found`/parse errors mark only that row; the batch continues.
- **429 mid-batch:** stop issuing requests, mark unresolved rows `pending`, return the partial table plus a `*RateLimitError`. Never discard already-earned results.

### D7 — Rate-limit signalling as a typed error
```go
type RateLimitError struct { Limit, Remaining int; ResetAt time.Time }
func (e *RateLimitError) Error() string
func (e *RateLimitError) Unwrap() error // returns ErrGeocodeRateLimited
```
Parsed from `X-Ratelimit-*`. Callers can `errors.Is(err, ErrGeocodeRateLimited)` for branching or read `ResetAt` for precise back-off. Sentinels: `ErrGeocodeNotFound`, `ErrGeocodeTimeout`, `ErrGeocodeRateLimited`, in `geocoding_errors.go` (mirrors `yfinance_errors.go`).

### D8 — Retry classification: transient only
Retry timeouts and transient network errors (up to `Retries`, backoff `RetryBackoff*(attempt+1)` like YFinance). Do **not** auto-retry `ErrGeocodeRateLimited` (cannot succeed within the window, wastes quota) or `ErrGeocodeNotFound` (deterministic).

### D9 — Optional cache: interface + two built-ins, definitive-only, exact key
```go
type GeocodeCache interface {
    Get(key string) (*ReverseGeocodeResult, bool)
    Set(key string, r *ReverseGeocodeResult)
}
```
`NewMemoryGeocodeCache()` (mutex-guarded map) and `NewFileGeocodeCache(path)` (JSON on disk, mutex-guarded, survives restarts — the highest-value option under a 15/hour cap). Key = `fmt.Sprintf("%.6f,%.6f", lat, lng)` (exact, never returns wrong data). Cache only success and `not_found`; never transient errors. `not_found` is represented in-cache by a distinguished marker so a cached miss short-circuits without a network call.

### D10 — Quota-safe tests
Unit tests run against `httptest.Server` returning canned 200/404/429/malformed bodies, driven through `BaseURL`. No live calls by default. Any real-endpoint smoke test sits behind a build tag (e.g. `//go:build geocoding_live`) so `go test ./...` never spends quota.

## Risks / Trade-offs

- **[15 req/hour makes large batches impractical without caching]** → dedup + opt-in `FileGeocodeCache` + partial-result-on-429 keep it usable; docs will state the limit plainly and recommend the file cache for repeated runs.
- **[Third-party endpoint may change or disappear]** → `BaseURL` override + isolated file + sentinel errors localize the blast radius; the `GoogleMapsStores` FIXME shows this class of breakage is real, so the fetcher degrades to typed errors rather than panics.
- **[429 body/shape is assumed, not yet observed live]** → probing it live would burn scarce quota; implementation keys off the HTTP 429 status and `X-Ratelimit-*` headers (already observed) rather than the body, and treats a missing header defensively (zero-value fields).
- **[`FileGeocodeCache` concurrent access]** → guard the in-memory map and file writes with a mutex; document that batch calls may run concurrently.
- **[Exact-key cache has low hit-rate for jittery GPS data]** → accepted deliberately; rounding keys risks wrong answers near boundaries. Users who want coarser keys can round coordinates before calling.

## Open Questions

- Should `Interval` default to a non-zero value to gently self-throttle toward the 15/hour budget? Leaning **no** — a forced multi-minute default would surprise interactive/demo use; leave 0 and document the quota. Revisit if users report accidental exhaustion.
- File-cache format: flat JSON map keyed by the coordinate string is the initial choice; revisit only if it proves too large in practice.
