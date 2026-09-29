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
dt.ToCSV("reviews.csv")
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
    dt.ToCSV("apple_store_reviews.csv")
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
  - **`YFHistoryParams` fields**:
    - `Period`: Time range: `"1d"`, `"5d"`, `"1mo"`, `"3mo"`, `"6mo"`, `"1y"`, `"2y"`, `"5y"`, `"10y"`, `"ytd"` or `"max"`.
    - `Interval`: Bar size: `"1m"`, `"2m"`, `"5m"`, `"15m"`, `"30m"`, `"60m"`, `"90m"`, `"1h"`, `"1d"`, `"5d"`, `"1wk"`, `"1mo"` or `"3mo"`.
    - `Start`, `End` (`*time.Time`): A date range to fetch instead of `Period`.
    - `PrePost`: Include pre- and post-market bars.
    - `AutoAdjust`: Adjust open, high, low and close for splits and dividends.
    - `Actions`: Include dividend and split events.
    - `Repair`: Repair bad data, such as prices off by 100x and missing values. `RepairOptions` (`*YFRepairOptions`) chooses which repairs run through `FixUnitMixups`, `FixZeroes`, `FixSplits`, `FixDividends` and `FixCapitalGains`; `nil` runs all of them.
    - `KeepNA`: Keep rows whose values are missing.
  - `YFHistoryParams` and `YFRepairOptions` are declared by `datafetch` itself, so upgrading the go-yfinance backend does not change them.
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

#### 5. News & Calendar

- `News(count int, tab YFNewsTab)`: Up to `count` news articles (`count <= 0` means 10). `tab` is `YFNewsTabNews` for news articles, which the empty tab also means, `YFNewsTabAll` for articles and press releases, or `YFNewsTabPressReleases` for press releases only. Any other value is an error, returned before any request.
- `Calendar()`: Upcoming earnings and dividend dates.

#### 6. Holders & Insider Trading

- `MajorHolders()`: Major holders percentages.
- `InstitutionalHolders()`: Detailed list of institutional holders.
- `MutualFundHolders()`: Detailed list of mutual fund holders.
- `InsiderTransactions()`: Records of insider trading activities.

#### 7. Analyst Estimates & Recommendations

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

// A DataTable's columns, each given as a column selector.
func (g *TWGeocodingClient) ReverseTable(dt *insyra.DataTable, latCol, lngCol any) (*insyra.DataTable, error)
```

`ReverseTable` picks each column the way every Insyra method does: a string is an Excel-style index (`"A"`, `"B"`, ...), `insyra.Name("lat")` is a column name, and an int is a 0-based position, counting from the end when negative. A bare string is never read as a name, so `ReverseTable(dt, "lat", "lng")` is an error, returned before any request, that tells you to write `insyra.Name("lat")`. `ReverseTableByColName(dt, "lat", "lng")` is deprecated; it is the same call as `ReverseTable(dt, insyra.Name("lat"), insyra.Name("lng"))` and is removed in the release after the one that deprecated it.

Batch semantics tuned for the 15/hour quota:
- **De-duplication** — identical `(lat, lng)` pairs cost at most one request.
- **Per-row tolerance** — a `not_found` or invalid coordinate marks only that row; the batch continues.
- **Partial results on quota exhaustion** — on HTTP 429 the fetcher stops, returns every row resolved so far with the rest marked `pending`, and returns a `*RateLimitError`.

```go
g, _ := datafetch.TWGeocoding(datafetch.TWGeocodingConfig{
    Cache: datafetch.NewFileGeocodeCache("geocache.json"),
})

enriched, err := g.ReverseTable(dt, insyra.Name("lat"), insyra.Name("lng"))
if rl := (*datafetch.RateLimitError)(nil); errors.As(err, &rl) {
    fmt.Printf("quota hit; resolved rows kept, resets at %s\n", rl.ResetAt)
}
enriched.Show()
```

### Cancellation

`ReverseContext`, `ReverseColsContext` and `ReverseTableContext` take a `context.Context` first; `Reverse`, `ReverseCols` and `ReverseTable` call them with `context.Background()`. When the context is done, the `Interval` throttle, the request in flight and any retry backoff stop at once, and the call returns `ctx.Err()`. A request the caller's deadline cut off is not retried, and is reported as `context.DeadlineExceeded` rather than `ErrGeocodeTimeout`, which means the service did not answer within `Timeout`.

A batch stopped part-way keeps what it resolved, the same way it does when the quota runs out: the table comes back with every row resolved so far, the rest of the valid coordinates marked `pending`, and `ctx.Err()`. No request is sent after the context is done, so the rows already paid for are not lost and the batch can be resumed from the `pending` rows. A `nil` context is an error.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
enriched, err := g.ReverseTableContext(ctx, dt, insyra.Name("lat"), insyra.Name("lng"))
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

`NewFileGeocodeCache` is the highest-value option under the 15/hour cap — resolved coordinates persist between program runs. Each write goes to a temporary file that is then renamed over the cache, so an interrupted write cannot corrupt it. Keys are the coordinate quantized to 6 decimal places (`"%.6f,%.6f"`, ~0.1 m — far finer than GPS accuracy), so a cache hit returns the correct region in practice.

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

### Methods and columns

```go
DailyPrices(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)
DailyPricesAdjusted(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)
ExRights(from, to time.Time, market TWMarket) (*insyra.DataTable, error)
InstitutionalTrades(date time.Time, market TWMarket) (*insyra.DataTable, error)
MarginBalance(date time.Time, market TWMarket) (*insyra.DataTable, error)
AllDailyQuotes(market TWMarket) (*insyra.DataTable, error)
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

`--`, `X`, and blank numeric cells become `nil`. TPEx historical prices report volume in lots and turnover in thousands of currency units, so `DailyPrices` converts them to shares and currency units to match the TWSE schema. This unit conversion is based on the source field names `成交張數` and `成交仟元`; margin balances retain the source endpoint's units.

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
returns := prices.PctChangeCol(insyra.Name("AdjClose"), 1).ClearNils()
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
