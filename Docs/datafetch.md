# [ datafetch ] Package

The `datafetch` package provides tools for retrieving data from external sources and converting them into Insyra data structures. Currently supports fetching store reviews from Google Maps (requires network access).

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
func GoogleMapsStores() *googleMapsStoreCrawler
```

**Description:** Creates a new Google Maps store crawler instance.

**Parameters:**

- None.

**Returns:**

- A crawler instance (unexported type), or `nil` if initialization fails

**Example:**

```go
crawler := datafetch.GoogleMapsStores()
if crawler == nil {
    log.Fatal("Failed to initialize crawler")
}
```

### Search

```go
func (c *googleMapsStoreCrawler) Search(query string) []GoogleMapsStoreData
```

**Description:** Searches for stores by name or keyword.

**Parameters:**

- `query`: Search keyword (store name, type, or location)

**Returns:**

- `[]GoogleMapsStoreData`: List of matching stores

**Example:**

```go
stores := crawler.Search("Starbucks Tokyo")
for _, store := range stores {
    fmt.Printf("Store: %s (ID: %s)\n", store.Name, store.ID)
}
```

### GetReviews

```go
func (c *googleMapsStoreCrawler) GetReviews(storeID string, pageCount int, options ...GoogleMapsStoreReviewsFetchingOptions) GoogleMapsStoreReviews
```

**Description:** Fetches reviews for a specific store.

**Parameters:**

- `storeID`: The store's Google Maps ID (obtained from Search)
- `pageCount`: Number of review pages to fetch (`0` fetches all available pages)
- `options`: Optional fetching configuration

**Returns:**

- `GoogleMapsStoreReviews`: Collection of reviews (can be converted to DataTable)

**Example:**

```go
// Basic usage (fetch one page)
reviews := crawler.GetReviews(store.ID, 1)

// With options
options := datafetch.GoogleMapsStoreReviewsFetchingOptions{
    SortBy:                          datafetch.SortByNewest,
    MaxWaitingInterval_Milliseconds: 3000,
}
reviews := crawler.GetReviews(store.ID, 20, options)
```

### ToDataTable

```go
func (r GoogleMapsStoreReviews) ToDataTable() *insyra.DataTable
```

**Description:** Converts reviews to an Insyra DataTable for analysis.

**Parameters:**

- None.

**Returns:**

- `*insyra.DataTable`: Table containing review data with columns:
  - `Reviewer`: Reviewer's display name
  - `ReviewerID`: Unique reviewer identifier
  - `ReviewerState`: Reviewer's location (if available)
  - `ReviewerLevel`: Local Guide level
  - `ReviewTime`: Time description (e.g., "2 weeks ago")
  - `ReviewDate`: Raw date string from the source
  - `Content`: Review text
  - `Rating`: Star rating (1-5)

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
    Reviewer      string    // Reviewer's display name
    ReviewerID    string    // Unique reviewer identifier
    ReviewerState string    // Reviewer's location
    ReviewerLevel int       // Local Guide level (0-10)
    ReviewTime    string    // Relative time (e.g., "2 weeks ago")
    ReviewDate    string    // Raw review date string
    Content       string    // Review text
    Rating        int       // Star rating (1-5)
}
```

### GoogleMapsStoreReviewsFetchingOptions

Configuration for review fetching.

```go
type GoogleMapsStoreReviewsFetchingOptions struct {
    SortBy                          GoogleMapsStoreReviewSortBy
    MaxWaitingInterval_Milliseconds uint
}
```

**Fields:**

- `SortBy`: How to sort reviews (default: by relevance)
- `MaxWaitingInterval_Milliseconds`: Maximum wait time between requests (helps avoid rate limiting)

### GoogleMapsStoreReviewSortBy

Review sorting options.

```go
const (
    SortByRelevance     GoogleMapsStoreReviewSortBy = 1 // Most relevant first (default)
    SortByNewest        GoogleMapsStoreReviewSortBy = 2 // Most recent first
    SortByHighestRating GoogleMapsStoreReviewSortBy = 3 // 5-star reviews first
    SortByLowestRating  GoogleMapsStoreReviewSortBy = 4 // 1-star reviews first
)
```

## Notes

- This crawler depends on Google Maps internal endpoints and a remote config file; availability can change without notice.
- Be prepared for rate limits or empty results and handle `nil` returns.
- Review fetching requires a stable internet connection.
- Large review counts may take longer to fetch.
- Use `MaxWaitingInterval_Milliseconds` to control request pacing.
- Store IDs are in the format `0x...:0x...`.

## Complete Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/HazelnutParadise/insyra/datafetch"
)

func main() {
    // Initialize crawler
    crawler := datafetch.GoogleMapsStores()
    if crawler == nil {
        log.Fatal("Failed to initialize crawler")
    }

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
        SortBy:                          datafetch.SortByNewest,
        MaxWaitingInterval_Milliseconds: 2000,
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

## Method Chaining

For concise code, you can chain method calls:

```go
// One-liner to get reviews as DataTable
dt := datafetch.GoogleMapsStores().
    GetReviews(
        datafetch.GoogleMapsStores().Search("Starbucks")[0].ID,
        1,
    ).
    ToDataTable()
```
