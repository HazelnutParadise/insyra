## 1. Errors & types

- [ ] 1.1 Create `datafetch/geocoding_errors.go` with sentinels `ErrGeocodeNotFound`, `ErrGeocodeTimeout`, `ErrGeocodeRateLimited` (mirror `yfinance_errors.go`).
- [ ] 1.2 Add `RateLimitError{Limit, Remaining int; ResetAt time.Time}` with `Error()` and `Unwrap() error` returning `ErrGeocodeRateLimited`.
- [ ] 1.3 Add a `classifyGeocodeError` / header-parsing helper that maps HTTP status + `X-Ratelimit-*` headers + `ok:false` bodies to the sentinels/`RateLimitError`, and a `retryable()` that returns true only for timeout/transient network errors.

## 2. Core fetcher & single lookup

- [ ] 2.1 In `datafetch/geocoding.go`, define `TWGeocodingConfig` (`Timeout, Interval, UserAgent, Retries, RetryBackoff, BaseURL, Cache`) and `normalize()` with defaults (Timeout 15s, default endpoint, default UA) and validation (reject negative Interval/Retries/RetryBackoff).
- [ ] 2.2 Define `ReverseGeocodeResult` (Lat, Lng + nine API fields) and its `ToDataTable()`.
- [ ] 2.3 Implement `TWGeocoding(cfg) (*twGeocoder, error)`: build `http.Client` with timeout, wire `limiter.IntervalLimiter` from `Interval`, store config + cache.
- [ ] 2.4 Implement `Reverse(lat, lng float64) (*ReverseGeocodeResult, error)`: cache check → limiter wait → GET `BaseURL?lat&lng` with UA → decode via `goccy/go-json` → map 200/404/429/malformed to result/sentinels → retry transient per `Retries`/`RetryBackoff` → write success/`not_found` to cache.

## 3. Batch lookups

- [ ] 3.1 Implement `ReverseCols(lat, lng *insyra.DataList) (*insyra.DataTable, error)`: length check, dedup identical coordinates, resolve each unique point via `Reverse`, fan results back to all rows, build a `DataTable` with region columns + `GeocodeStatus`.
- [ ] 3.2 On a mid-batch `*RateLimitError`, stop further requests, mark unresolved rows `pending`, return the partial table plus the rate-limit error.
- [ ] 3.3 Implement `ReverseTable(dt, latCol, lngCol string)` resolving columns by Excel index (via `GetCol`) and delegating to `ReverseCols`; return an error on missing/invalid columns.
- [ ] 3.4 Implement `ReverseTableByColName(dt, latColName, lngColName string)` resolving via `GetColByName` and delegating to `ReverseCols`; return an error on missing columns.

## 4. Optional cache

- [ ] 4.1 Define `GeocodeCache` interface (`Get(key) (*ReverseGeocodeResult, bool)`, `Set(key, r)`) and the exact `"%.6f,%.6f"` key helper; represent cached `not_found` with a distinguished marker.
- [ ] 4.2 Implement `NewMemoryGeocodeCache()` (mutex-guarded map).
- [ ] 4.3 Implement `NewFileGeocodeCache(path)` (mutex-guarded JSON map, load on construct, persist on `Set`); ensure only success/`not_found` are ever stored.

## 5. Tests (quota-safe)

- [ ] 5.1 Create `datafetch/geocoding_test.go` with an `httptest.Server` fixture serving canned 200 / 404 not_found / 429 (+`X-Ratelimit-*`) / malformed-JSON responses, wired through `BaseURL`.
- [ ] 5.2 Cover single lookup: success field mapping, `not_found`→`ErrGeocodeNotFound`, empty `village_eng`, 429→`RateLimitError` + `errors.Is(ErrGeocodeRateLimited)`, timeout→`ErrGeocodeTimeout`, retry-then-succeed, no-retry-on-rate-limit.
- [ ] 5.3 Cover batch: dedup issues one request for duplicates, per-row `not_found` tolerance, index vs. name column addressing equivalence, invalid column error, 429 mid-batch returns partial table + error.
- [ ] 5.4 Cover cache: hit avoids network, `not_found` cached, transient errors not cached, `FileGeocodeCache` persists across instances (temp dir).
- [ ] 5.5 Add a build-tag-gated (`//go:build geocoding_live`) smoke test hitting the real endpoint; excluded from default `go test ./...`.

## 6. Docs & skills sync (same change)

- [ ] 6.1 Add a Geocoding section to `Docs/datafetch.md`: quick start, `TWGeocoding`/`Reverse`/batch/cache API reference, the 15 req/hour limit, and error handling.
- [ ] 6.2 Update `skills/insyra/` (`SKILL.md` and/or `references/`) so datafetch coverage includes TW reverse geocoding.
- [ ] 6.3 Confirm `README.md` / `README_TW.md` package tables need no new row (datafetch already listed) and `allpkgs/allpkgs.go` already imports `datafetch`.

## 7. Verify

- [ ] 7.1 `go build ./...` and `go test ./datafetch/...` green; `golangci-lint run` clean on new files.
- [ ] 7.2 `openspec verify --change add-datafetch-tw-geocoding` (or `/opsx:verify`) — implementation matches proposal/specs/design.
