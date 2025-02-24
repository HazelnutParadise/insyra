# [ datafetch ] Package

`datafetch` is a data retrieval toolkit designed to simplify the process of fetching and handling various types of data. Currently, it supports fetching store reviews from Google Maps and converting them into an `insyra.DataTable`.

## Installation

Use `go get` to install the `datafetch` package:

```sh
go get github.com/HazelnutParadise/insyra/datafetch
```

## Fetching Store Reviews from Google Maps

### Initialize Google Maps Store Crawler

```go
import "github.com/HazelnutParadise/insyra/datafetch"

crawler := datafetch.GoogleMapsStores()
if crawler == nil {
    log.Fatal("Failed to initialize Google Maps crawler")
}
```

### Search for Stores

```go
stores := crawler.Search("Din Tai Fung")
for _, store := range stores {
    fmt.Printf("ID: %s, Name: %s\n", store.ID, store.Name)
}
```

### Fetch Store Reviews

```go
reviews := crawler.GetReviews("0x123456789abcdef:0xfedcba987654321", 5)
if reviews == nil {
    log.Fatalf("Error fetching reviews")
}

for _, review := range reviews {
    fmt.Printf("Reviewer: %s, Rating: %d, Review: %s\n", review.Reviewer, review.Rating, review.Content)
}
```

### Using Review Fetching Options

You can customize review fetching options using `GoogleMapsStoreReviewsFetchingOptions`. For example:

```go
options := datafetch.GoogleMapsStoreReviewsFetchingOptions{
    SortBy:             datafetch.SortByNewest,
    MaxWaitingInterval: 3000,
}

reviews := crawler.GetReviews("0x123456789abcdef:0xfedcba987654321", 5, options)
if reviews == nil {
    log.Fatalf("Error fetching reviews")
}
```

### Chained Calls

You can use method chaining to fetch store reviews and convert them into a `DataTable`, for example:

```go
crawler := datafetch.GoogleMapsStores()
dt := crawler.GetReviews(crawler.Search("Din Tai Fung")[0].ID, 5).ToDataTable()
if dt == nil {
    log.Fatalf("Error")
}

dt.Show()
```

### Convert Reviews to DataTable or CSV

```go
dt := reviews.ToDataTable() // Convert reviews to DataTable
csv := dt.ToCSV("reviews.csv", false, true) // Convert DataTable to CSV
```

## Structs and Functions

### `GoogleMapsStoreReview`

Represents a Google Maps store review.

```go
type GoogleMapsStoreReview struct {
    Reviewer      string    `json:"reviewer"`
    ReviewerID    string    `json:"reviewer_id"`
    ReviewerState string    `json:"reviewer_state"`
    ReviewerLevel int       `json:"reviewer_level"`
    ReviewTime   string    `json:"review_time"`
    ReviewDate   time.Time `json:"review_date"`
    Content       string    `json:"content"`
    Rating        int       `json:"rating"`
}
```

### `GoogleMapsStoreReviewsFetchingOptions`

Options for fetching reviews.

```go
type GoogleMapsStoreReviewsFetchingOptions struct {
    SortBy                          GoogleMapsStoreReviewSortBy
    MaxWaitingInterval_Milliseconds uint
}
```

### `GoogleMapsStoreReviewSortBy`

Sorting options for reviews.

```go
const (
    SortByRelevance      GoogleMapsStoreReviewSortBy = 1
    SortByNewest         GoogleMapsStoreReviewSortBy = 2
    SortByHighestRating  GoogleMapsStoreReviewSortBy = 3
    SortByLowestRating   GoogleMapsStoreReviewSortBy = 4
)
```
