# [ datafetch ] Package

The `datafetch` package provides utilities for retrieving data from external sources and converting them into Insyra data structures (for example, `*insyra.DataTable`). It currently includes a Google Maps store review crawler and a Yahoo Finance wrapper (returns `*insyra.DataTable`). Network access is required for remote fetchers and some features depend on third-party backends which may change.

## Installation

```bash
go get github.com/HazelnutParadise/insyra/datafetch
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/HazelnutParadise/insyra/datafetch"
)

func main() {
    // Initialize the crawler
    crawler := datafetch.GoogleMapsStores()

    // Search for stores
    stores := crawler.Search("Din Tai Fung")
    if len(stores) == 0 {
        fmt.Println("No stores found")
        return
    }

    // Get reviews for the first store
    // pageCount is the number of review pages to fetch (0 = all available)
    reviews := crawler.GetReviews(stores[0].ID, 1)

    // Convert to DataTable for analysis
    dt := reviews.ToDataTable()
    dt.Show()
}
```

## API Reference

### GoogleMapsStores

```go
func GoogleMapsStores() *GoogleMapsStoresClient
```

**Description:** Creates a new Google Maps store crawler instance.

**Parameters:**

- None.

**Returns:**

- A `*GoogleMapsStoresClient`. Creating it needs no network access, and it is never `nil`. The type can be named in your own structs and signatures, but its zero value is not a working client: `Search` and `GetReviews` on a `GoogleMapsStoresClient` that did not come from `GoogleMapsStores` send nothing and return `nil` with a warning.

**Example:**

```go
crawler := datafetch.GoogleMapsStores()
```

### Search

```go
func (c *GoogleMapsStoresClient) Search(query string) []GoogleMapsStoreData
```

**Description:** Searches for stores by name or keyword.

**Parameters:**

- `query`: Search keyword (store name, type, or location)

**Returns:**

- `[]GoogleMapsStoreData`: Up to 20 matching stores, in Google's order. `nil` when the request fails or no store comes back; a warning says which.

**Example:**

```go
stores := crawler.Search("Starbucks Tokyo")
for _, store := range stores {
    fmt.Printf("Store: %s (ID: %s)\n", store.Name, store.ID)
}
```

### GetReviews

```go
func (c *GoogleMapsStoresClient) GetReviews(storeID string, pageCount int, options ...GoogleMapsStoreReviewsFetchingOptions) GoogleMapsStoreReviews
```

**Description:** Fetches reviews for a specific store, 10 per page, from the review window Google Search shows for the store. No sign-in is needed.

**Parameters:**

- `storeID`: The store's Google Maps ID (obtained from Search)
- `pageCount`: Number of review pages to fetch, 10 reviews each (`0` fetches every page)
- `options`: Optional fetching configuration

**Returns:**

- `GoogleMapsStoreReviews`: Collection of reviews (can be converted to DataTable). `nil` when a request fails or Google's response format has changed; a warning says which.

**Example:**

```go
// Basic usage (fetch one page)
reviews := crawler.GetReviews(store.ID, 1)

// With options
options := datafetch.GoogleMapsStoreReviewsFetchingOptions{
    SortBy:             datafetch.GoogleMapsStoreReviewSortByNewest,
    MaxWaitingInterval: 3 * time.Second,
}
reviews := crawler.GetReviews(store.ID, 20, options)
```

### SearchContext and GetReviewsContext

```go
func (c *GoogleMapsStoresClient) SearchContext(ctx context.Context, query string) []GoogleMapsStoreData
func (c *GoogleMapsStoresClient) GetReviewsContext(ctx context.Context, storeID string, pageCount int, options ...GoogleMapsStoreReviewsFetchingOptions) GoogleMapsStoreReviews
```

**Description:** `Search` and `GetReviews` with a `context.Context`; the plain methods call them with `context.Background()`. When `ctx` is done, the request in flight or the wait between review pages stops at once, and the method returns `nil` with a warning, the way it reports any other failure. Check `ctx.Err()` to tell a cancellation from a failed request. A `nil` context is refused the same way.

**Example:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()
reviews := crawler.GetReviewsContext(ctx, store.ID, 0)
if reviews == nil && ctx.Err() != nil {
    fmt.Println("stopped:", ctx.Err())
}
```

### ToDataTable

```go
func (r GoogleMapsStoreReviews) ToDataTable() *insyra.DataTable
```

**Description:** Converts reviews to an Insyra DataTable for analysis.

**Parameters:**

- None.

**Returns:**

- `*insyra.DataTable`: Table containing review data, one row per review. The columns are ordered by name, in this order:
  - `Content`: Review text
  - `Language`: The review's language code, such as `zh-Hant`. Empty for a review with only a star rating, which has no text
  - `Rating`: Star rating (1-5)
  - `ReviewDate`: The date the review was posted, in UTC, as `YYYY-MM-DD`
  - `ReviewID`: The review's own ID, useful for dropping repeats when merging pages fetched in different sort orders
  - `ReviewTime`: Time description as Google shows it, in Traditional Chinese (e.g., "2 個月前")
  - `Reviewer`: Reviewer's display name
  - `ReviewerID`: Unique reviewer identifier
  - `ReviewerLevel`: Always 0; Google's review pages no longer include it
  - `ReviewerPhotoCount`: How many photos the reviewer has posted
  - `ReviewerReviewCount`: How many reviews the reviewer has written
  - `ReviewerState`: Always empty; Google's review pages no longer include it

**Example:**

```go
dt := reviews.ToDataTable()
dt.Show()
dt.ToCSV("reviews.csv", false, true, false)
```

## Data Types

### GoogleMapsStoreData

Represents a store from search results.

```go
type GoogleMapsStoreData struct {
    ID   string // Unique store identifier
    Name string // Store name
}
```

### GoogleMapsStoreReview

Represents a single review.

```go
type GoogleMapsStoreReview struct {
    Reviewer            string // Reviewer's display name
    ReviewerID          string // Unique reviewer identifier
    ReviewerState       string // Always empty; no longer provided by Google
    ReviewerLevel       int    // Always 0; no longer provided by Google
    ReviewerReviewCount int    // How many reviews the reviewer has written
    ReviewerPhotoCount  int    // How many photos the reviewer has posted
    ReviewID            string // The review's own ID
    ReviewTime          string // Relative time in Traditional Chinese (e.g., "2 個月前")
    ReviewDate          string // Posting date in UTC, YYYY-MM-DD
    Language            string // The review's language code, such as zh-Hant; empty when there is no text
    Content             string // Review text
    Rating              int    // Star rating (1-5)
}
```

### GoogleMapsStoreReviewsFetchingOptions

Configuration for review fetching.

```go
type GoogleMapsStoreReviewsFetchingOptions struct {
    SortBy                          GoogleMapsStoreReviewSortBy
    MaxWaitingInterval              time.Duration
    MaxWaitingInterval_Milliseconds uint // Deprecated: use MaxWaitingInterval
}
```

**Fields:**

- `SortBy`: How to sort reviews. Zero means by relevance.
- `MaxWaitingInterval`: The longest wait between two review pages, which helps avoid rate limiting. Each wait is random between one second and this value, so it must be at least one second; a smaller value is replaced by the default with a warning. Zero means 5 seconds.
- `MaxWaitingInterval_Milliseconds`: **Deprecated.** The same limit counted in milliseconds, removed in the release after the one that deprecated it. Setting it together with `MaxWaitingInterval` is an error: `GetReviews` returns `nil` with a warning before any request.

### GoogleMapsStoreReviewSortBy

Review sorting options.

```go
const (
    GoogleMapsStoreReviewSortByRelevance     GoogleMapsStoreReviewSortBy = 1 // Most relevant first (default)
    GoogleMapsStoreReviewSortByNewest        GoogleMapsStoreReviewSortBy = 2 // Most recent first
    GoogleMapsStoreReviewSortByHighestRating GoogleMapsStoreReviewSortBy = 3 // 5-star reviews first
    GoogleMapsStoreReviewSortByLowestRating  GoogleMapsStoreReviewSortBy = 4 // 1-star reviews first
)
```

`SortByRelevance`, `SortByNewest`, `SortByHighestRating` and `SortByLowestRating` are deprecated constants with the same values, removed in the release after the one that deprecated them.

## Notes

- This crawler depends on internal Google endpoints: the Google Maps result list for `Search`, and the review window on Google Search results for `GetReviews`, because Google Maps shows a signed-out visitor only five reviews. Availability can change without notice.
- Be prepared for rate limits or empty results and handle `nil` returns.
- Every request times out after 30 seconds. Progress is logged at debug level.
- Review fetching requires a stable internet connection.
- Large review counts may take longer to fetch.
- Use `MaxWaitingInterval` to control request pacing.
- Store IDs are in the format `0x...:0x...`.

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/HazelnutParadise/insyra/datafetch"
)

func main() {
    // Initialize crawler
    crawler := datafetch.GoogleMapsStores()

    // Search for stores
    stores := crawler.Search("Apple Store Taipei")
    if len(stores) == 0 {
        log.Fatal("No stores found")
    }

    fmt.Printf("Found %d stores\n", len(stores))
    for i, store := range stores {
        fmt.Printf("  %d. %s\n", i+1, store.Name)
    }

    // Fetch reviews for the first store with custom options
    options := datafetch.GoogleMapsStoreReviewsFetchingOptions{
        SortBy:             datafetch.GoogleMapsStoreReviewSortByNewest,
        MaxWaitingInterval: 2 * time.Second,
    }

    reviews := crawler.GetReviews(stores[0].ID, 2, options)
    if reviews == nil {
        log.Fatal("Failed to fetch reviews")
    }

    // Convert to DataTable
    dt := reviews.ToDataTable()
    rows, _ := dt.Size()
    fmt.Printf("\nFetched %d reviews\n", rows)

    // Display first 5 reviews
    dt.ShowRange(5)

    // Export to CSV
    dt.ToCSV("apple_store_reviews.csv", false, true, false)
    fmt.Println("\nReviews exported to apple_store_reviews.csv")
}
```

## Yahoo Finance (yfinance)

The `datafetch` package provides a lightweight, Python-like wrapper for Yahoo Finance data. It adapts the public API to return `*insyra.DataTable`, making results ready for immediate analysis and visualization.

### Quick Start

```go
package main

import (
    "time"
    "github.com/HazelnutParadise/insyra/datafetch"
)

func main() {
    // 1. Initialize the fetcher
    yf, _ := datafetch.YFinance(datafetch.YFinanceConfig{
        Timeout: 10 * time.Second,
    })

    // 2. Fetch historical data (as DataTable) using chained calls
    history, _ := yf.Ticker("AAPL").History(datafetch.YFHistoryParams{
        Period:   "1mo",
        Interval: "1d",
    })
    history.Show()
}
```

---

### Initialization & Configuration

#### YFinance

```go
func YFinance(cfg YFinanceConfig) (*YFinanceClient, error)
```

Creates a stateful fetcher, a `*YFinanceClient`. The type can be named in your own structs and signatures; a `YFinanceClient` or `YFTicker` that did not come from `YFinance` and `Ticker`, such as a zero value, returns an error from every method instead of fetching.

**YFinanceConfig Fields:**

| Field          | Type            | Description                                                                  | Default              |
| :------------- | :-------------- | :--------------------------------------------------------------------------- | :------------------- |
| `Timeout`      | `time.Duration` | Per-request timeout limit.                                                   | `15s`                |
| `Interval`     | `time.Duration` | Minimum spacing between the scheduled starts of successive requests (for rate limiting); no request starts before its scheduled time. Set to `0` to disable. | `0`                  |
| `UserAgent`    | `string`        | HTTP User-Agent header. See [How the client identifies itself](#how-the-client-identifies-itself). | Chrome 117 on Windows |
| `Retries`      | `int`           | Number of retry attempts on failure.                                         | `0`                  |
| `RetryBackoff` | `time.Duration` | Base backoff duration between retries.                                       | `300ms`              |

#### Ticker

```go
func (y *YFinanceClient) Ticker(symbol string) *YFTicker
```

Returns a `*YFTicker` bound to the fetcher. This method is designed to support chained calls (e.g., `yf.Ticker("AAPL").History(...)`). Any errors encountered during initialization or method calls are returned by the subsequent action methods.

---

### Ticker Methods

After obtaining a ticker object via `yf.Ticker(symbol)`, the following methods are available. Most methods return `*insyra.DataTable`.

#### 1. History & Quotes

- `History(params YFHistoryParams)`: Fetches historical OHLCV bars.
  - **Key `YFHistoryParams` fields**:
    - `Period`: Time range (e.g., `"1d", "5d", "1mo", "1y", "max"`).
    - `Interval`: Data granularity (e.g., `"1m", "5m", "1d", "1wk"`).
    - `Start`, `End`: Specific date range (format `YYYY-MM-DD`).
- `Quote()`: Fetches current quote summary.
- `FastInfo()`: Returns quick statistics (Market Cap, Price, etc.).
- `Info()`: Returns comprehensive company/security metadata.

#### 2. Corporate Actions & Dividends

- `Dividends()`: Historical dividend payments.
- `Splits()`: Historical stock split records.
- `Actions()`: Combined dividends and splits history.

#### 3. Financial Statements

These methods return a `*datafetch.YFFinancialStatementTables` structure containing three DataTables: `Values`, `Items`, and `Meta`.

- `IncomeStatement(freq YFPeriod)`: Income statement data.
- `BalanceSheet(freq YFPeriod)`: Balance sheet data.
- `CashFlow(freq YFPeriod)`: Cash flow statement data.

**YFPeriod Options:**

- `datafetch.YFPeriodAnnual` (Default)
- `datafetch.YFPeriodQuarterly`

`datafetch.YFPeriodYearly` is deprecated: it fetches the same statements as `YFPeriodAnnual` but labels the tables `yearly`, and it is removed in the release after the one that deprecated it.

#### 4. Options & Derivatives

- `Options()`: Returns a list of available expiration dates.
- `OptionChain(date string)`: Fetches the option chain for a specific date.
  - Returns `*datafetch.YFOptionChainTables` containing: `Calls`, `Puts`, `Underlying` (as DataTables) and `Expiration` (as time.Time).

#### 5. Holders & Insider Trading

- `MajorHolders()`: Major holders percentages.
- `InstitutionalHolders()`: Detailed list of institutional holders.
- `MutualFundHolders()`: Detailed list of mutual fund holders.
- `InsiderTransactions()`: Records of insider trading activities.

#### 6. Analyst Estimates & Recommendations

- `Recommendations()`: Analyst rating suggestions.
- `AnalystPriceTargets()`: Analyst price targets.
- `EarningsEstimate()`: Earnings per share estimates.
- `RevenueEstimate()`: Revenue estimates.
- `GrowthEstimates()`: Growth projections.
- `EPSTrend() / EPSRevisions()`: Trends and revisions in EPS.

---

### Cancellation

Every ticker method that requests data has a `...Context` form taking a `context.Context` first, such as `HistoryContext(ctx, params)`, `QuoteContext(ctx)`, `OptionChainContext(ctx, date)` and `NewsContext(ctx, count, tab)`; the plain method calls it with `context.Background()`. `Earnings`, `Sustainability`, `FundsData` and `TopHoldings` send no request and have none.

When the context is done, the method returns `ctx.Err()` at once, so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` tells a cancellation from a failure. A call whose context is already done sends nothing, and `History` and `Quote` stop waiting for the `Interval` throttle or a retry backoff. Once a call has started, though, the Yahoo Finance backend offers no way to stop it, and one call can make several requests: a cookie, a crumb, then the data. Such a call is abandoned rather than interrupted. It runs to its end in the background and may still send its remaining requests, each limited only by the backend's own timeout, which a stalled connection can outlast, and its result is discarded. Until it ends it keeps using the client, so a later call on the same client may have to wait for it. A `nil` context is an error.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
history, err := yf.Ticker("AAPL").HistoryContext(ctx, datafetch.YFHistoryParams{Period: "1y", Interval: "1d"})
if errors.Is(err, context.DeadlineExceeded) {
    // Yahoo did not answer within five seconds.
}
```

### Notes & Limitations

1. **Rate Limiting**: Frequent requests may lead to temporary IP blocks by Yahoo Finance. Use the `Interval` setting in `YFinanceConfig` to mitigate this.
2. **Automatic Date Conversion**: `datafetch` automatically attempts to convert columns named `Date`, `Time`, `Expiry`, etc., into Go `time.Time` objects for easier filtering and plotting.
3. **Unsupported Methods**: Due to underlying backend library limitations, the following methods return a "not supported" error:
   - `Earnings()` (Full earnings reports)
   - `Sustainability()` (ESG scores)
   - `FundsData()`, `TopHoldings()` (Fund-specific data)

### How the client identifies itself

By default the Yahoo Finance client presents itself to Yahoo as a web browser, in two ways:

- It sends the `User-Agent` header of Chrome 117 on Windows.
- The go-yfinance backend opens every connection with Chrome's TLS fingerprint, whatever the header says.

Yahoo's terms of service apply to how you use the data. To send a different header, set `YFinanceConfig.UserAgent`; the TLS fingerprint cannot be changed through `datafetch`.

## Taiwan Reverse Geocoding (TWGeocoding)

`TWGeocoding` wraps the [geocoding.zuola.com](https://geocoding.zuola.com/) reverse-geocoding API, turning a `(lat, lng)` coordinate into its Taiwan administrative region (county / town / village). It follows the same config-driven, stateful pattern as `YFinance`.

> **Reverse only.** The service maps coordinates → regions. It does **not** do forward geocoding (address → coordinates).
>
> **Free-tier quota: 15 requests/hour per IP.** This is the dominant constraint. Use a `GeocodeCache` and the batch methods (which de-duplicate identical coordinates) to make the budget last. See [Handling the rate limit](#handling-the-rate-limit).

### Quick Start

```go
package main

import (
    "errors"
    "fmt"

    "github.com/HazelnutParadise/insyra/datafetch"
)

func main() {
    g, err := datafetch.TWGeocoding(datafetch.TWGeocodingConfig{})
    if err != nil {
        panic(err)
    }

    res, err := g.Reverse(24.9884079, 121.4598882)
    switch {
    case err == nil:
        fmt.Printf("%s%s%s\n", res.CountyName, res.TownName, res.VillageName) // 新北市土城區青雲里
    case errors.Is(err, datafetch.ErrGeocodeNotFound):
        fmt.Println("point is outside any Taiwan village")
    default:
        fmt.Println("lookup failed:", err)
    }
}
```

### Initialization & Configuration

```go
func TWGeocoding(cfg TWGeocodingConfig) (*TWGeocodingClient, error)
```

**`TWGeocodingConfig` fields:**

| Field          | Type            | Description                                                            | Default             |
| :------------- | :-------------- | :-------------------------------------------------------------------- | :------------------ |
| `Timeout`      | `time.Duration` | Per-request timeout.                                                  | `15s`               |
| `Interval`     | `time.Duration` | Minimum spacing between the scheduled starts of successive requests (client-side throttle); no request starts before its scheduled time. `0` = off. | `0`                 |
| `UserAgent`    | `string`        | HTTP User-Agent header.                                               | (browser-like UA)   |
| `Retries`      | `int`           | Retry attempts for **transient** failures (timeout / network).        | `0`                 |
| `RetryBackoff` | `time.Duration` | Base backoff between retries (`backoff * (attempt+1)`).               | `300ms`             |
| `BaseURL`      | `string`        | Endpoint override (for mocks/tests or a future paid/self-hosted tier).| Official endpoint   |
| `Cache`        | `GeocodeCache`  | Optional result cache. `nil` disables caching.                        | `nil`               |

Invalid values (negative `Interval` / `Retries` / `RetryBackoff`) return an error. The returned `*TWGeocodingClient` can be named in your own structs and signatures; a `TWGeocodingClient` that did not come from `TWGeocoding`, such as a zero value, returns an error from every method instead of sending a request.

### Single Lookup

```go
func (g *TWGeocodingClient) Reverse(lat, lng float64) (*ReverseGeocodeResult, error)
```

Returns a typed `*ReverseGeocodeResult`; `ErrGeocodeNotFound` when the point is outside any village; a `*RateLimitError` when the quota is exhausted; `ErrGeocodeTimeout` on timeout.

```go
type ReverseGeocodeResult struct {
    Lat, Lng    float64
    VillCode    string // e.g. "65000130032"
    CountyName  string // 新北市
    TownName    string // 土城區
    VillageName string // 青雲里
    VillageEng  string // Qingyun Vil. (may be empty for some villages)
    CountyID, CountyCode, TownID, TownCode string
}

func (r *ReverseGeocodeResult) ToDataTable() *insyra.DataTable
```

### Batch Lookups

Batch methods reverse-geocode many coordinates and return an `*insyra.DataTable` with the original coordinates, the resolved region columns, and a `GeocodeStatus` column (`ok` / `not_found` / `pending` / `invalid_coordinate` / `error: ...`).

```go
// Two parallel DataLists.
func (g *TWGeocodingClient) ReverseCols(lat, lng *insyra.DataList) (*insyra.DataTable, error)

// A DataTable's columns, addressed by Excel-style index ("A", "B", ...).
func (g *TWGeocodingClient) ReverseTable(dt *insyra.DataTable, latCol, lngCol string) (*insyra.DataTable, error)

// A DataTable's columns, addressed by name.
func (g *TWGeocodingClient) ReverseTableByColName(dt *insyra.DataTable, latColName, lngColName string) (*insyra.DataTable, error)
```

Batch semantics tuned for the 15/hour quota:
- **De-duplication** — identical `(lat, lng)` pairs cost at most one request.
- **Per-row tolerance** — a `not_found` or invalid coordinate marks only that row; the batch continues.
- **Partial results on quota exhaustion** — on HTTP 429 the fetcher stops, returns every row resolved so far with the rest marked `pending`, and returns a `*RateLimitError`.

```go
g, _ := datafetch.TWGeocoding(datafetch.TWGeocodingConfig{
    Cache: datafetch.NewFileGeocodeCache("geocache.json"),
})

enriched, err := g.ReverseTableByColName(dt, "lat", "lng")
if rl := (*datafetch.RateLimitError)(nil); errors.As(err, &rl) {
    fmt.Printf("quota hit; resolved rows kept, resets at %s\n", rl.ResetAt)
}
enriched.Show()
```

### Cancellation

`ReverseContext`, `ReverseColsContext` and `ReverseTableContext` take a `context.Context` first; `Reverse`, `ReverseCols` and `ReverseTable` call them with `context.Background()`. `ReverseTableByColName` has no Context form: pass the same columns to `ReverseTableContext` by their Excel-style index. When the context is done, the `Interval` throttle, the request in flight and any retry backoff stop at once, and the call returns `ctx.Err()`. A request the caller's deadline cut off is not retried, and is reported as `context.DeadlineExceeded` rather than `ErrGeocodeTimeout`, which means the service did not answer within `Timeout`.

A batch stopped part-way keeps what it resolved, the same way it does when the quota runs out: the table comes back with every row resolved so far, the rest of the valid coordinates marked `pending`, and `ctx.Err()`. No request is sent after the context is done, so the rows already paid for are not lost and the batch can be resumed from the `pending` rows. A `nil` context is an error.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
enriched, err := g.ReverseTableContext(ctx, dt, "A", "B")
if errors.Is(err, context.DeadlineExceeded) {
    // enriched holds the rows resolved in time; the rest are "pending".
}
```

### Optional Caching

Since village boundaries are effectively static, results are highly cacheable. A cache stores only **definitive** outcomes (successful results and `not_found`), never transient failures, so nothing wrong is ever served.

```go
type GeocodeCache interface {
    Get(key string) (*ReverseGeocodeResult, bool)
    Set(key string, r *ReverseGeocodeResult)
}

func NewMemoryGeocodeCache() GeocodeCache          // in-process, concurrency-safe
func NewFileGeocodeCache(path string) GeocodeCache // JSON file; survives across runs
```

`NewFileGeocodeCache` is the highest-value option under the 15/hour cap — resolved coordinates persist between program runs. Keys are the coordinate quantized to 6 decimal places (`"%.6f,%.6f"`, ~0.1 m — far finer than GPS accuracy), so a cache hit returns the correct region in practice.

### Errors

| Error / Type            | Meaning                                                          |
| :---------------------- | :--------------------------------------------------------------- |
| `ErrGeocodeNotFound`    | Coordinate is outside any Taiwan village (sea / outside Taiwan). |
| `ErrGeocodeTimeout`     | Request exceeded `Timeout`.                                      |
| `ErrGeocodeRateLimited` | Quota exhausted (sentinel).                                      |
| `*RateLimitError`       | Carries `Limit`, `Remaining`, `ResetAt`; unwraps to `ErrGeocodeRateLimited`. |

### Handling the rate limit

The free tier allows **15 requests/hour per IP**, reported via `X-Ratelimit-*` response headers and enforced with HTTP 429.

- Match `errors.Is(err, datafetch.ErrGeocodeRateLimited)` to detect exhaustion, and read `*RateLimitError.ResetAt` to know when to retry.
- Rate-limit errors are **not** auto-retried (a retry within the same window cannot succeed and would waste quota).
- Prefer batch methods (de-dup) plus a `NewFileGeocodeCache` for repeated or large workloads.

### Notes

- Depends on the third-party `geocoding.zuola.com` endpoint; availability and quota are outside this library's control. Override `BaseURL` for a self-hosted or paid tier.
- Some villages have an empty `village_eng`; this is returned as an empty string, not an error.

## Taiwan Stock Exchanges (TWStock)

`datafetch.TWStock` reads unauthenticated daily data from the Taiwan Stock Exchange (TWSE) and Taipei Exchange (TPEx) into typed `*insyra.DataTable` values. It does not require an API key and makes no request until a fetch method is called.

### Quick Start

```go
package main

import (
    "log"
    "time"

    "github.com/HazelnutParadise/insyra/datafetch"
)

func main() {
    stocks, err := datafetch.TWStock(datafetch.TWStockConfig{Interval: 300 * time.Millisecond})
    if err != nil {
        log.Fatal(err)
    }
    prices, err := stocks.DailyPrices(
        "2330", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
        time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), datafetch.TWMarketTWSE,
    )
    if err != nil {
        log.Fatal(err)
    }
    prices.Show()
}
```

Use `TWMarketTWSE`, `TWMarketTPEx`, or `TWMarketAuto`. `Auto` tries TWSE first and falls back to TPEx when the exchange reports no data. The returned `Market` column records the exchange used for daily prices.

### Configuration

```go
func TWStock(cfg TWStockConfig) (*TWStockClient, error)
```

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `Timeout` | `time.Duration` | `15s` | Per-request timeout. |
| `Interval` | `time.Duration` | `0` | Minimum spacing between the scheduled starts of successive requests; no request starts before its scheduled time. `0` disables throttling. |
| `UserAgent` | `string` | `insyra-datafetch/<version>` | HTTP User-Agent header. |
| `Retries` | `int` | `0` | Retry attempts for HTTP, timeout, network, JSON, and exchange-payload errors. |
| `RetryBackoff` | `time.Duration` | `300ms` | Base delay before a retry, multiplied by attempt number. |
| `Concurrency` | `int` | `6` | Normalized concurrency setting reserved for batched fetch extensions. |

Negative `Interval`, `Retries`, `RetryBackoff`, or `Concurrency` values return an error. Non-2xx responses, invalid JSON, and exchange errors are never silently swallowed. The returned `*TWStockClient` can be named in your own structs and signatures; a `TWStockClient` that did not come from `TWStock`, such as a zero value, returns an error from every method instead of sending a request.

**Being throttled.** The TWSE site (and the MOPS pages) sometimes answers HTTP 307 with a "for security reasons the page cannot be shown" page and no `Location`, and TPEx or the OpenAPI hosts may answer 429. The error then matches `errors.Is(err, datafetch.ErrTWStockThrottled)` and its text says the source refused the request as too frequent. It is retried like any failed request, but the defaults (`Retries` 0) give up at the first one, which ends a multi-year backfill. Reports from 2026-10-17 saw one URL answer 307, 307, 200 when requested every 3 seconds, so for a backfill start from something like `Interval: 3 * time.Second`, `Retries: 5`, `RetryBackoff: 3 * time.Second` and widen them until the run completes. These values are a starting point, not a measured guarantee; the defaults are unchanged.

### Methods and columns

```go
DailyPrices(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)
DailyPricesAdjusted(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)
ExRights(from, to time.Time, market TWMarket) (*insyra.DataTable, error)
InstitutionalTrades(date time.Time, market TWMarket) (*insyra.DataTable, error)
MarginBalance(date time.Time, market TWMarket) (*insyra.DataTable, error)
AllDailyQuotes(market TWMarket) (*insyra.DataTable, error)
OddLotQuotes(date time.Time, session TWOddLotSession, market TWMarket) (*insyra.DataTable, error)
ShareholdingDistribution() (*insyra.DataTable, error)
MonthlyRevenue(year, month int, market TWMarket) (*insyra.DataTable, error)
DailyQuotesOn(date time.Time, market TWMarket) (*insyra.DataTable, error)
AdvanceDecline(date time.Time, market TWMarket, commonStockOnly bool) (*insyra.DataTable, error)
```

Each method has a `...Context` form taking a `context.Context` first, such as `DailyPricesContext(ctx, code, from, to, market)`; the plain method calls it with `context.Background()`. When the context is done, the `Interval` throttle, the request in flight and any retry backoff stop at once, and the call returns `ctx.Err()` without sending another request, so a long backfill can be stopped between two monthly requests. A `nil` context is an error.

`DailyPrices` requests one month at a time, filters to the inclusive `[from, to]` range, and sorts by `Date` ascending.

| Method | Columns |
| :--- | :--- |
| `DailyPrices` | `Date` (`time.Time`), `Code` (`string`), `Volume` (`int64`, shares), `Turnover` (`int64`, currency units), `Open`, `High`, `Low`, `Close`, `Change` (`float64`), `Transactions` (`int64`), `Market` (`string`) |
| `DailyPricesAdjusted` | every `DailyPrices` column, plus `AdjFactor`, `AdjOpen`, `AdjHigh`, `AdjLow`, `AdjClose` (`float64`) |
| `ExRights` | `Date` (`time.Time`, the ex-date), `Code`, `Name` (`string`), `PrevClose`, `RefPrice`, `Distribution` (`float64`), `Kind` (`string`), `AdjFactor` (`float64`) |
| `InstitutionalTrades` | `Date` (`time.Time`), `Code` (`string`), `Name` (`string`), `ForeignNet`, `TrustNet`, `DealerNet`, `TotalNet` (`int64`, shares) |
| `MarginBalance` | `Date` (`time.Time`), `Code` (`string`), `Name` (`string`), `MarginBalance`, `ShortBalance` (`int64`, source balance units) |
| `AllDailyQuotes` | `Date` (`time.Time`), `Code` (`string`), `Name` (`string`), `Volume` (`int64`, shares), `Turnover` (`int64`, currency units), `Open`, `High`, `Low`, `Close`, `Change` (`float64`), `Transactions` (`int64`) |
| `MonthlyRevenue` | see Monthly revenue below |
| `DailyQuotesOn` | the columns of `AllDailyQuotes` plus `Sign` (`string`): `"+"` up, `"-"` down, `"="` unchanged, `"X"` no reference price (then `Change` is `nil`) |
| `AdvanceDecline` | one row: `Date` (`time.Time`), `Advancing`, `AdvancingLimit`, `Declining`, `DecliningLimit`, `Unchanged`, `NoTrade`, `NoReference` (`int64`), `Source` (`string`) |
| `OddLotQuotes` | `Date` (`time.Time`), `Code`, `Name` (`string`), `Volume` (`int64`, shares), `Transactions`, `Turnover` (`int64`), `BidPrice`, `AskPrice` (`float64`), `BidVolume`, `AskVolume` (`int64`), plus `Open`, `Close`, `High`, `Low` (`float64`) for `TWOddLotIntraday` or `Price` for `TWOddLotAfterHours` |

`--`, `X`, and blank numeric cells become `nil`. TPEx historical prices report volume in lots and turnover in thousands of currency units, so `DailyPrices` converts them to shares and currency units to match the TWSE schema. This unit conversion is based on the source field names `成交張數` and `成交仟元`; margin balances retain the source endpoint's units.

### Any day's closes and advancing and declining counts

`AllDailyQuotes` gives only the latest day. `DailyQuotesOn(date, market)` gives every security's close for any past day, one request per day: from the TWSE's `MI_INDEX` report (data since 2004-02-11, including securities delisted since, so a study of the whole market has no survivorship bias) or the TPEx's daily close report. `Change` is signed and measured from the exchange's own reference price, which already accounts for ex-rights and ex-dividend days; `Sign` carries the exchange's up or down mark. A weekend or holiday gives an empty table with the columns in place. `TWMarketAuto` tries the TWSE and then the TPEx.

`AdvanceDecline(date, market, commonStockOnly)` counts the securities that rose, fell and stood still. For recent TWSE days the exchange publishes the count itself (漲跌證券數合計) and that is returned with `Source` `"official"`, including the limit-up and limit-down counts: its 股票 column when `commonStockOnly`, the whole-market column otherwise. Without that table, and always for the TPEx, the counts are made from the closes' marks with `Source` `"computed"` and the limit counts `nil`; `commonStockOnly` then keeps four-digit codes starting with 1 to 9 and drops those starting with 91, the depositary receipts. On 2026-10-16 the computed 684 up and 286 down equal the official stock column.

The TWSE answers repeated requests with the throttling page described above; for a multi-year backfill set `Interval` and `Retries` as suggested there.

### Odd-lot quotes

`OddLotQuotes(date, session, market)` returns a market's odd-lot report (trades of fewer than 1,000 shares) for one day, one row per security. `TWMarketTWSE` reads `TWTC7U` (intraday, `TWOddLotIntraday`, available since 2020-10-26) or `TWT53U` (after hours, `TWOddLotAfterHours`, since 2005); a price the exchange shows as `--` is `nil` and its 合計 totals row is left out. `TWMarketTPEx` reads the TPEx (上櫃) pages `afterTrading/oddQuote` (intraday, since 2020-10-26) and `afterTrading/odd` (after hours, since January 2011); its intraday report's change column is left out and a price shown as `0.00` (no trade or no quote) is `nil`. The two markets give the same columns, so the tables can be stacked. As in `DailyQuotesOn`, `TWMarketAuto` reads the TWSE and only if it has no data for the date the TPEx. Prices are as traded and not adjusted for ex-rights. A holiday, or a date before the report existed, gives an empty table with the columns in place, not an error. The TPEx's OpenAPI (`tpex_odd_stock`) gives only the latest day and is not used. The TWSE answers repeated requests with the throttling page described above, so for a backfill set `Interval` and `Retries` as suggested there.

### Monthly revenue

`MonthlyRevenue(year, month, market)` returns the monthly operating revenue of every listed company for one month from the Market Observation Post System (MOPS) summary pages, one page per market, which saves the two thousand per-company requests. `year` is the Gregorian year (2025, not 114), the pages start in January 2003, and `TWMarketAuto` returns both TWSE and TPEx. A month with no page gives an empty table with the columns in place.

| Column | Meaning |
| :--- | :--- |
| `Month` (`time.Time`) | the first day of the month the revenue is for |
| `Code`, `Name`, `Industry`, `Market` (`string`) | the company; `Market` is `"TWSE"` or `"TPEx"` |
| `Revenue`, `PrevMonthRevenue`, `LastYearRevenue` (`int64`) | this month, last month and the same month a year earlier, in **thousands of NT dollars** |
| `MoMChange`, `YoYChange` (`float64`) | percent change from last month and from a year earlier, as the site computes them |
| `Cumulative`, `LastYearCumulative` (`int64`) | the year to date and the year before's, in thousands of NT dollars |
| `CumulativeChange` (`float64`) | percent change of the year to date |
| `Remark` (`string`) | the company's note, empty when there is none |
| `PageDate` (`time.Time`) | when the page was produced |

**`PageDate` is not an announcement date.** It is the date the whole page was produced (出表日期), not the day each company published its figure; companies report by the 10th of the following month. Do not use it as the moment a figure was known and could be traded on. Names are decoded from Big5, which loses a few rare characters.

Newer months have a domestic page and a second page for companies listed from abroad, and both are read; older months have a single page. The MOPS answers repeated requests with the throttling page described above, so for a backfill set `Interval` and `Retries` as suggested there.

### Shareholding distribution

`ShareholdingDistribution()` returns the Taiwan Depository & Clearing Corporation's weekly shareholding distribution, from `opendata.tdcc.com.tw`: for every security, how many holders hold how many shares at each holding level. It is the usual way to watch big holders against small ones, and it exists only for Taiwan.

| Column | Meaning |
| :--- | :--- |
| `Date` (`time.Time`) | the data date of the week |
| `Code` (`string`) | security code |
| `Level` (`int64`), `LevelLabel` (`string`) | the holding level, 1 to 17 |
| `Holders`, `Shares` (`int64`) | holders and shares at that level |
| `Percent` (`float64`) | the share of the securities held in custody |

Levels 1 to 15 are ranges of shares held: 1-999, 1,000-5,000, 5,001-10,000, 10,001-15,000, 15,001-20,000, 20,001-30,000, 30,001-40,000, 40,001-50,000, 50,001-100,000, 100,001-200,000, 200,001-400,000, 400,001-600,000, 600,001-800,000, 800,001-1,000,000 and 1,000,001 and above. Level 16 is the difference adjustment the TDCC publishes, and level 17 the total. About 69,000 rows come back in one request, about 2.4 MB.

**Only the latest week is available.** The TDCC publishes no history through this source, so save each week's table if you want a series; the earlier weeks need the TDCC website's per-security query or a paid source.

### Adjusted prices and corporate actions

On an ex-dividend or ex-rights day the quoted price drops by the distribution without any loss to the holder, so a return series built from raw `Close` shows a fake loss on every ex-date. `DailyPricesAdjusted` removes it.

```go
prices, err := stocks.DailyPricesAdjusted(
    "2330", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
    time.Now(), datafetch.TWMarketTWSE,
)
if err != nil {
    log.Fatal(err)
}
returns := prices.PctChangeCol("AdjClose", 1).ClearNils()
```

- The factor is the exchange's own 除權息參考價 ÷ 除權息前收盤價, taken from `ExRights`, so it already includes rights and the tax-free portion and needs no dividend arithmetic.
- Adjustment is **backward**, the same convention as Yahoo Finance's `Adj Close`: every bar strictly before an ex-date is multiplied by that ex-date's factor, factors compound across multiple ex-dates, and the last bar keeps the quoted price (`AdjFactor` is `1` from the last ex-date in range onwards).
- Ex-dates are fetched for `[from, to]` only. A distribution after `to` is not applied, so two calls with different `to` produce different adjusted histories — extend `to` to today for a stable series.
- `Volume`, `Turnover`, and `Change` are not adjusted. A `nil` raw price yields a `nil` adjusted price.
- `AdjClose` and Yahoo's `Adj Close` are both backward-adjusted ratios but not identical: Yahoo builds its factor from cash dividends and splits, while the TWSE reference price also folds in rights. Small differences are expected.

`ExRights` returns the exchange's 除權除息計算結果表 for the range, sorted by `Date` then `Code`. `Kind` is `"dividend"` (息), `"rights"` (權), or `"both"` (權息). Ranges longer than a year are split into contiguous one-year requests and merged. `AdjFactor` is `nil` when either reference price is unreadable.

**TPEx is not supported by `ExRights` or `DailyPricesAdjusted`.** TPEx publishes only a current-announcement list, with no dated ex-rights history endpoint, so both methods return an explicit "TPEx ex-rights not supported" error for `TWMarketTPEx` rather than a silently empty table. `TWMarketAuto` queries TWSE only for these two methods.

### Data source, paging, and limits

- TWSE sources are the [TWSE OpenAPI](https://openapi.twse.com.tw/) and its dated legacy JSON endpoints. TWSE OpenAPI data is subject to the [Taiwan Government Open Data License](https://data.gov.tw/en/license).
- TPEx sources are its legacy JSON endpoints and [TPEx OpenAPI](https://www.tpex.org.tw/openapi/). Usage is subject to [TPEx's trading-information terms](https://eshop.tpex.org.tw/en/product/shoppingTerm).
- A ten-year per-stock history is approximately 120 monthly requests. Set `Interval` to at least `300ms` for backfills and tune retries for transient 5xx responses.
- `ExRights` splits ranges longer than a year into contiguous one-year requests, so a ten-year ex-rights history is ten requests.
- The exchange table is whatever the exchange serves at call time; nothing is cached, so a reference price the exchange revises afterwards changes later results.
- The exchange rate limit is not published as a stable contract. Intraday, real-time quotes, order books, fundamentals, dividend payout details beyond the ex-date reference prices, and local caching are outside this client.

## Taiwan Futures Exchange (TAIFEX)

```go
client, err := datafetch.TAIFEX(datafetch.TWStockConfig{})
```

`TAIFEX` reads the Taiwan Futures Exchange's open data (`openapi.taifex.com.tw`), which needs no key. It takes the same settings as `TWStock` (`Timeout`, `Interval`, `UserAgent`, `Retries`, `RetryBackoff`) and shares its throttling and retry behaviour. A `TAIFEXClient` that did not come from `TAIFEX`, such as a zero value, returns an error from every method. Each method has a `...Context` form.

```go
InstitutionalFutures() (*insyra.DataTable, error)
InstitutionalOptions() (*insyra.DataTable, error)
LargeTradersFutures() (*insyra.DataTable, error)
LargeTradersOptions() (*insyra.DataTable, error)
PutCallRatio() (*insyra.DataTable, error)
```

**Only the latest day.** The OpenAPI gives the latest trading day for the first four tables, and the last 20 trading days for `PutCallRatio`. The `...History` methods below read longer histories from the exchange's website CSV downloads.

| Method | Columns |
| :--- | :--- |
| `InstitutionalFutures` | `Date` (`time.Time`), `Contract`, `Trader` (`string`; 自營商 dealers, 投信 investment trusts, 外資及陸資 foreign investors), then `TradeLong`, `TradeShort`, `TradeNet` and `OILong`, `OIShort`, `OINet` (`int64`, contracts), each with a `...Value` column (`int64`, thousands of NT dollars) |
| `InstitutionalOptions` | the same, plus `CallPut` (`"CALL"` or `"PUT"`) after `Contract` |
| `LargeTradersFutures` | `Date`, `Contract`, `ContractName`, `SettlementMonth`, `TraderType` (`string`), `Top5Buy`, `Top5Sell`, `Top10Buy`, `Top10Sell`, `MarketOI` (`int64`, contracts) |
| `LargeTradersOptions` | the same, plus `CallPut` (`"買權"` call or `"賣權"` put) after `ContractName` |
| `PutCallRatio` | `Date`, `PutVolume`, `CallVolume`, `PutOI`, `CallOI` (`int64`), `VolumeRatio`, `OIRatio` (`float64`, percent: 88.75 means the put volume is 88.75% of the call volume) |

In the large-trader tables, `Top5Buy` and the others are the open long and short positions held by the 5 and 10 largest traders, and `MarketOI` is the whole market's open interest. `SettlementMonth` is kept as the exchange publishes it, a string such as `"202610"`, where `"999912"` is the total over all months. `TraderType` is `"0"` for all traders and `"1"` for the specific institutional traders. The exchange answers some of its tables as CSV with Chinese headers whatever is asked for; the client reads either form into the same columns.

### History from the website

```go
InstitutionalFuturesHistory(from, to time.Time) (*insyra.DataTable, error)
InstitutionalOptionsHistory(from, to time.Time) (*insyra.DataTable, error)
LargeTradersFuturesHistory(from, to time.Time) (*insyra.DataTable, error)
LargeTradersOptionsHistory(from, to time.Time) (*insyra.DataTable, error)
PutCallRatioHistory(from, to time.Time) (*insyra.DataTable, error)
```

Each returns every trading day from `from` to `to` in the columns of the matching method above, oldest first (`PutCallRatio` gives newest first). They read the CSV downloads of `www.taifex.com.tw` (Big5 encoded), month by month, since the exchange limits how long a range may be, so a long range makes many requests under the client's `Interval`; for a multi-year range set `Interval` and `Retries` as for `TWStock`.

How far back each table goes was measured on 2026-10-20 and is the exchange's, not the library's: the two institutional tables only about the last 2.4 years (from 2024), the large-trader tables to about 2004 for futures and to about 2005 for options, and the put/call ratio to about 2003. A date the website refuses makes the call return an error that quotes the website's message and the range, never an empty table; a range with no trading day (a weekend) is an empty table. A contract's code and settlement month are trimmed of the padding the website puts around them.

## FinMind

```go
client, err := datafetch.FinMind(datafetch.FinMindConfig{Token: os.Getenv("FINMIND_TOKEN")})
prices, err := client.Dataset("TaiwanStockPrice", "2330",
    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Time{})
```

[FinMind](https://finmindtrade.com) serves Taiwan and US market data, one security's whole history per request: daily prices from 1994, monthly revenue, financial statements, dividends, delisted securities' prices and more, over 100 datasets (the list is at <https://finmind.github.io/llms-full.txt>). It complements `TWStock`, which reads the exchanges' own pages one day for the whole market: FinMind suits a long history of one security, `TWStock` a whole market one day at a time.

`FinMind` takes a `FinMindConfig`: `Token` (sent in an `Authorization` header; leave it empty for the free tier without a token, or when a proxy adds it), `Timeout`, `Interval`, `UserAgent`, `Retries` and `RetryBackoff`. A `FinMindClient` that did not come from `FinMind` returns an error from every method.

```go
Dataset(name, dataID string, from, to time.Time) (*insyra.DataTable, error)
```

`Dataset` is the one entry point, with a `DatasetContext` form, and it returns the dataset's columns as FinMind sends them, in FinMind's order and under FinMind's names: integers are `int64`, other numbers `float64`, text stays text, `null` is `nil`, and a column named `date` becomes a `time.Time` when every value is a date. `dataID` is the security (`"2330"`); most datasets need one on the free tier. A zero `from` or `to` leaves that end open. A dataset with no rows gives an empty table with no columns.

**Errors you can tell apart.** `errors.Is(err, datafetch.ErrFinMindLimit)` means the plan's request limit was reached (HTTP 402); `errors.Is(err, datafetch.ErrFinMindLevel)` means the plan does not allow the request, such as a free account asking for a dataset above its level, or one that needs a `dataID` without giving it. Neither is retried, since waiting or upgrading is the fix, not repeating the call. Other failures, such as an unknown dataset (HTTP 422, whose message lists the valid names), are ordinary errors.

**Traps in the data.** One is checked: the `date` of `TaiwanStockMonthRevenue` is the first day of the month *after* the revenue month (`revenue_month` 7 has `date` 2026-08-01) and is not the announcement date, so aligning it with prices on `date` looks ahead. The rest were reported when the dataset was tried on 2026-10-17 and are not checked here: individual institutional trades start on 2012-05-02, not 2005 as the documentation says; a daily price row with no trade has all four prices 0, so filter with `close > 0`; and the free tier refuses adjusted prices (`TaiwanStockPriceAdj`) though the documentation says it allows them.

## Japan Exchange Group (JPX)

```go
client, err := datafetch.JPX(datafetch.TWStockConfig{Interval: 3 * time.Second, Retries: 2})
weeks, err := client.InvestorTypes(
    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
    datafetch.JPXUnitValue)
```

The JPX publishes every week how much each type of investor sold and bought on the Tokyo and Nagoya markets: individuals (cash and margin separately), foreigners, investment trusts, business companies, financial institutions, proprietary trading and more (投資部門別売買状況). It is the one public weekly series in a major market that gives individuals' actual purchases and sales. `JPX` takes the same settings as `TWStock` (`Timeout`, `Interval`, `UserAgent`, `Retries`, `RetryBackoff`); one file per week is a request, so for a year or more set an `Interval` of about 3 seconds, which was enough to fetch every week since 2016. A `JPXClient` that did not come from `JPX`, such as a zero value, returns an error from every method.

```go
InvestorTypes(from, to time.Time, unit JPXUnit) (*insyra.DataTable, error)
```

with an `InvestorTypesContext` form. It returns the weeks that overlap `[from, to]` from 2016, as one row per week, market and investor group:

| Column | Meaning |
| :--- | :--- |
| `WeekStart`, `WeekEnd` (`time.Time`) | the first and last trading day of the week, so a holiday week is one to three days and a week can cross the year end |
| `Market` (`string`) | `"TSE Prime"`, `"TSE Standard"`, `"TSE Growth"`, `"Tokyo & Nagoya"`; for weeks before April 2022 `"TSE 1st"`, `"TSE 2nd"`, `"TSE Mothers"`, `"TSE JASDAQ"` and `"Tokyo & Nagoya"` |
| `Investor` (`string`) | one of fourteen groups that do not overlap and add up to the market: `Proprietary Cash`, `Proprietary Margin`, `Individuals Cash`, `Individuals Margin`, `Foreigners Institutions`, `Foreigners Individuals`, `Securities Cos.`, `Investment Trusts`, `Business Cos.`, `Other Cos.`, `Life & Non-Life`, `City & Regional BK`, `Trust BK`, `Other Financials` |
| `Sales`, `Purchases`, `Balance`, `Total` (`int64`) | sales, purchases, purchases less sales, and sales plus purchases, in **thousands of yen** for `JPXUnitValue` or **thousands of shares** for `JPXUnitShares` |

Individuals as a whole are the two Individuals rows summed, and likewise for foreigners and proprietary trading.

**Publication lag.** A week's figures come out on the fourth business day of the following week at 15:00 JST, so a backtest must lag them by at least that; `WeekEnd` is not the date the data was known.

**Two file formats.** Weeks since September 2026 are one workbook per week (`.xlsx`). Earlier weeks are `.xls` files, one for value and one for shares, that hold the previous week beside the report week; only the report week of each is used, and the cash and margin rows are cross-checked against the blocks they split (a mismatch logs a warning). The yearly archive pages are searched for the years in the range and the year after it. The file names carry random IDs, so they are found on the pages, not built. The `.xls` files are read by a small reader inside the package, since the available libraries dropped cells of these files. A week the JPX has not published, or a range with no files, gives no rows.
