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
comments := crawler.GetComments("0x123456789abcdef:0xfedcba987654321", 5)
if comments == nil {
    log.Fatalf("Error fetching reviews")
}

for _, comment := range comments {
    fmt.Printf("Reviewer: %s, Rating: %d, Comment: %s\n", comment.Reviewer, comment.Rating, comment.Content)
}
```

### Using Review Fetching Options

You can customize review fetching options using `GoogleMapsStoreCommentsFetchingOptions`. For example:

```go
options := datafetch.GoogleMapsStoreCommentsFetchingOptions{
    SortBy:             datafetch.SortByNewest,
    MaxWaitingInterval: 3000,
}

comments := crawler.GetComments("0x123456789abcdef:0xfedcba987654321", 5, options)
if comments == nil {
    log.Fatalf("Error fetching reviews")
}
```

### Chained Calls

You can use method chaining to fetch store reviews and convert them into a `DataTable`, for example:

```go
crawler := datafetch.GoogleMapsStores()
dt := crawler.GetComments(crawler.Search("Din Tai Fung")[0].ID, 5).ToDataTable()
if dt == nil {
    log.Fatalf("Error")
}

dt.Show()
```

### Convert Reviews to DataTable or CSV

```go
dt := comments.ToDataTable() // Convert comments to DataTable
csv := dt.ToCSV("comments.csv", false, true) // Convert DataTable to CSV
```

## Structs and Functions

### `GoogleMapsStoreComment`

Represents a Google Maps store review.

```go
type GoogleMapsStoreComment struct {
    Reviewer      string    `json:"reviewer"`
    ReviewerID    string    `json:"reviewer_id"`
    ReviewerState string    `json:"reviewer_state"`
    ReviewerLevel int       `json:"reviewer_level"`
    CommentTime   string    `json:"comment_time"`
    CommentDate   time.Time `json:"comment_date"`
    Content       string    `json:"content"`
    Rating        int       `json:"rating"`
}
```

### `GoogleMapsStoreCommentsFetchingOptions`

Options for fetching reviews.

```go
type GoogleMapsStoreCommentsFetchingOptions struct {
    SortBy             GoogleMapsStoreCommentSortBy
    MaxWaitingInterval uint
}
```

### `GoogleMapsStoreCommentSortBy`

Sorting options for reviews.

```go
const (
    SortByRelevance      GoogleMapsStoreCommentSortBy = 1
    SortByNewest         GoogleMapsStoreCommentSortBy = 2
    SortByHighestRating  GoogleMapsStoreCommentSortBy = 3
    SortByLowestRating   GoogleMapsStoreCommentSortBy = 4
)
```
