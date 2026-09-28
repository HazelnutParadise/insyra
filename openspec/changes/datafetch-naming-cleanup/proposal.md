# Proposal: datafetch-naming-cleanup

## Why

DF-5 of the API review, filed under [#212](https://github.com/HazelnutParadise/insyra/issues/212), found four places in `datafetch` where one thing has two names or a name breaks the package's own pattern:

- `YFPeriodAnnual` (`"annual"`) and `YFPeriodYearly` (`"yearly"`) ask Yahoo for the same statements; go-yfinance turns `"yearly"` into `"annual"` before the request. The only difference a caller sees is the frequency label written into the result tables' names and `Meta`.
- `GoogleMapsStoreReviewsFetchingOptions.MaxWaitingInterval_Milliseconds` is a `uint` count of milliseconds with an underscore in its name, where every other duration in the package, such as `TWStockConfig.Interval`, is a `time.Duration`.
- `TWGeocoding`'s `ReverseTable(dt, latCol, lngCol string)` reads its columns as Excel-style indexes and `ReverseTableByColName` reads them as names. Under the one-column-selector rule in `AGENTS.md` that is one operation with two entry points: the selector already says which kind it was given.
- The review sort orders are `SortByRelevance`, `SortByNewest`, `SortByHighestRating` and `SortByLowestRating`, while the package's other enumerations start with their type's name (`TWMarketTWSE`, `YFPeriodQuarterly`). `SortByRelevance` in a caller's code does not say it belongs to Google Maps reviews.

The owner ruled on #211 that each function has one name, and that an old name stays for one release as a Deprecated wrapper keeping its old meaning.

## What Changes

- `YFPeriodYearly` is **Deprecated** in favour of `YFPeriodAnnual`. It keeps its value, so a table fetched with it is still labelled `yearly` until it is removed.
- `GoogleMapsStoreReviewsFetchingOptions` gains `MaxWaitingInterval time.Duration`, with the same rules the millisecond field has: zero means 5 seconds, a value under one second is replaced by the default with a warning, and each wait between pages is random between one second and this value. `MaxWaitingInterval_Milliseconds` is **Deprecated** and keeps working. Setting both is refused with a warning and a `nil` result before any request, the way two options structs already are.
- `ReverseTable(dt, latCol, lngCol any)` takes each column as the library's selector: a string is an Excel-style index, `insyra.Name("lat")` is a name, an int is a position. A call passing two strings compiles and means what it meant. `ReverseTableByColName` is **Deprecated** and calls `ReverseTable` with `insyra.Name(...)`, which is what it did.
- The sort orders are `GoogleMapsStoreReviewSortByRelevance`, `…ByNewest`, `…ByHighestRating` and `…ByLowestRating`. The four old names are **Deprecated** constants with the same values.
- The removals are recorded as an `AGENTS.md` follow-up for the release after the one that ships this.

## Capabilities

### New Capabilities

- `datafetch-twgeocoding`: how `TWGeocoding`'s table method picks its columns.

### Modified Capabilities

- `datafetch-gmaps-crawler`: "Review fetching options and progress" names the `time.Duration` field and refuses both fields set; adds "Review sort orders carry their type's name".
- `datafetch-yfinance-api` (added by `datafetch-own-yfinance-types`): adds "One statement frequency has one name".

## Impact

- `datafetch/yfinance.go`, `datafetch/googleMapsCommentCrawler.go`, `datafetch/geocoding.go` and their tests. Nothing outside `datafetch` uses the renamed or deprecated names.
- `Docs/datafetch.md`, both changelogs, `AGENTS.md` (removal follow-up), `api-review.md` (DF-5), `delivery-status.md`.
