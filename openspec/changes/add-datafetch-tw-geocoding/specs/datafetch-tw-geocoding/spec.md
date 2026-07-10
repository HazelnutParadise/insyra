## ADDED Requirements

### Requirement: Construct a Taiwan reverse-geocoding fetcher

The `datafetch` package SHALL provide `TWGeocoding(cfg TWGeocodingConfig) (*twGeocoder, error)`, a stateful fetcher following the existing `YFinance` config-driven pattern. It SHALL normalize the config, applying defaults for unset fields, and SHALL return an error for invalid values rather than silently continuing.

#### Scenario: Default configuration
- **WHEN** `TWGeocoding` is called with a zero-value `TWGeocodingConfig`
- **THEN** it returns a usable fetcher whose request timeout defaults to 15 seconds, whose `BaseURL` defaults to the official `geocoding.zuola.com` reverse endpoint, and whose request interval is 0 (no client-side throttling)

#### Scenario: Invalid configuration rejected
- **WHEN** `TWGeocoding` is called with a negative `Interval`, `Retries`, or `RetryBackoff`
- **THEN** it returns a non-nil error and a nil fetcher

#### Scenario: Endpoint override
- **WHEN** `TWGeocodingConfig.BaseURL` is set to a custom URL
- **THEN** all requests are issued against that URL instead of the default endpoint

### Requirement: Single-point reverse geocoding

The fetcher SHALL provide `Reverse(lat, lng float64) (*ReverseGeocodeResult, error)` that returns a typed result containing the county, town, and village names, their English/official codes, and the village code for a coordinate that resolves to a Taiwan village.

#### Scenario: Successful lookup
- **WHEN** `Reverse` is called with a coordinate inside a Taiwan village and the API returns `ok:true`
- **THEN** it returns a `*ReverseGeocodeResult` with `CountyName`, `TownName`, `VillageName`, `VillCode`, and the related code fields populated, and a nil error

#### Scenario: Point outside any village
- **WHEN** the API responds with HTTP 404 and `{"ok":false,"error":"not_found"}` (e.g. a point at sea or outside Taiwan)
- **THEN** `Reverse` returns a nil result and an error satisfying `errors.Is(err, ErrGeocodeNotFound)`

#### Scenario: Missing English village name tolerated
- **WHEN** a successful response omits or has an empty `village_eng`
- **THEN** `Reverse` still returns a successful result with `VillageEng` set to the empty string, not an error

### Requirement: Rate-limit-aware error handling

The fetcher SHALL treat the free tier's 15-requests-per-hour quota as a first-class concern. It SHALL read the `X-Ratelimit-Limit`, `X-Ratelimit-Remaining`, and `X-Ratelimit-Reset` response headers, and on HTTP 429 SHALL return a `RateLimitError` carrying the parsed limit, remaining count, and reset time.

#### Scenario: Quota exhausted
- **WHEN** the API responds with HTTP 429
- **THEN** the returned error is a `*RateLimitError` whose `Limit`, `Remaining`, and `ResetAt` fields reflect the response headers

#### Scenario: Rate-limit error is identifiable
- **WHEN** a caller receives a `*RateLimitError`
- **THEN** `errors.Is(err, ErrGeocodeRateLimited)` returns true so callers can branch without depending on the concrete type

#### Scenario: Rate-limit errors are not auto-retried
- **WHEN** a request fails with a rate-limit error and `Retries` is greater than 0
- **THEN** the fetcher does NOT re-issue the request (retrying within the same window cannot succeed and would waste quota) and returns the rate-limit error immediately

### Requirement: Batch reverse geocoding over Insyra data structures

The fetcher SHALL provide batch methods that reverse-geocode many coordinates and return a `*insyra.DataTable`: `ReverseCols(lat, lng *insyra.DataList)`, `ReverseTable(dt *insyra.DataTable, latCol, lngCol string)` addressing columns by Excel-style index, and `ReverseTableByColName(dt *insyra.DataTable, latColName, lngColName string)` addressing columns by name. The result SHALL contain the resolved region columns plus a `GeocodeStatus` column describing each row's outcome.

#### Scenario: Enrich coordinate columns
- **WHEN** `ReverseCols` is given equal-length latitude and longitude `DataList`s
- **THEN** it returns a `DataTable` whose rows carry the original coordinates, the resolved county/town/village fields, and a `GeocodeStatus` value per row

#### Scenario: Address columns by Excel index vs. name
- **WHEN** `ReverseTable(dt, "A", "B")` and `ReverseTableByColName(dt, "lat", "lng")` are called on a table whose first two columns are named `lat` and `lng`
- **THEN** both resolve the same two coordinate columns and produce equivalent enriched tables

#### Scenario: Per-row failure tolerated
- **WHEN** a batch contains a coordinate that resolves to `not_found`
- **THEN** that row's region fields are left empty and its `GeocodeStatus` records the failure, while the remaining rows are still resolved (the batch does not abort)

#### Scenario: Duplicate coordinates deduplicated
- **WHEN** a batch contains N rows with an identical `(lat, lng)` pair
- **THEN** the fetcher issues at most one network request for that coordinate and reuses the result for all N rows

#### Scenario: Quota exhausted mid-batch
- **WHEN** the API returns HTTP 429 partway through a batch
- **THEN** the fetcher stops issuing further requests, returns a `DataTable` containing every row resolved so far with unresolved rows marked pending in `GeocodeStatus`, and returns a `*RateLimitError`

#### Scenario: Invalid column reference
- **WHEN** a batch method is given a missing column name/index or mismatched-length coordinate lists
- **THEN** it returns a nil table and a non-nil error

### Requirement: Optional result caching

The fetcher SHALL support an opt-in cache via a `GeocodeCache` interface, with built-in in-memory (`NewMemoryGeocodeCache`) and file-backed (`NewFileGeocodeCache`) implementations. A cache SHALL be keyed by the exact coordinate and SHALL store only definitive outcomes (successful results and `not_found`), never transient failures.

#### Scenario: Cache hit avoids a network request
- **WHEN** a coordinate has already been resolved and stored in the configured cache
- **THEN** a subsequent lookup of the same coordinate is served from the cache without issuing a network request or consuming quota

#### Scenario: Definitive not-found is cached
- **WHEN** a coordinate resolves to `not_found`
- **THEN** that outcome is cached, so re-querying the same point does not spend another request

#### Scenario: Transient failures are not cached
- **WHEN** a lookup fails with a rate-limit or timeout error
- **THEN** nothing is written to the cache, so the coordinate can be retried later

#### Scenario: File cache persists across fetcher instances
- **WHEN** a `NewFileGeocodeCache` backed by a path is populated and a new fetcher is later constructed with a cache on the same path
- **THEN** previously resolved coordinates are served from the file without new network requests

### Requirement: Timeout and retry behavior

Each request SHALL honor the configured `Timeout`. The fetcher SHALL retry only transient failures (request timeouts and transient network errors) up to `Retries` times with a backoff derived from `RetryBackoff`, and SHALL classify a timeout as `ErrGeocodeTimeout`.

#### Scenario: Request timeout
- **WHEN** a request exceeds the configured `Timeout`
- **THEN** the returned error satisfies `errors.Is(err, ErrGeocodeTimeout)`

#### Scenario: Retry then succeed
- **WHEN** a transient error occurs on the first attempt, `Retries` is at least 1, and the retry succeeds
- **THEN** the fetcher returns the successful result after honoring the backoff delay

#### Scenario: Non-retryable errors are returned immediately
- **WHEN** a request fails with `ErrGeocodeNotFound` or a rate-limit error
- **THEN** the fetcher returns without consuming remaining retry attempts
