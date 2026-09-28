## ADDED Requirements

### Requirement: Each constructor returns an exported type

`TWStock` SHALL return `*TWStockClient`, `YFinance` SHALL return `*YFinanceClient`, `(*YFinanceClient).Ticker` SHALL return `*YFTicker`, `TWGeocoding` SHALL return `*TWGeocodingClient` and `GoogleMapsStores` SHALL return `*GoogleMapsStoresClient`, each with the parameters and errors it has today. A program outside the module SHALL be able to name each of these types in a declaration.

#### Scenario: Naming the types from another package
- **WHEN** 在 `datafetch` 之外的套件宣告一個 struct，欄位型別分別是五個匯出型別，並把四個建構子與 `Ticker` 的回傳值存進去
- **THEN** 程式可以編譯

### Requirement: A client not built by its constructor fails without panicking

A zero value or a nil pointer of `TWStockClient`, `TWGeocodingClient`, `YFinanceClient` or `YFTicker` SHALL return an error from every method that fetches, naming the constructor to use, and SHALL NOT panic or send a request. A zero value or a nil pointer of `GoogleMapsStoresClient` SHALL return `nil` from `Search` and `GetReviews` with a warning naming `GoogleMapsStores`, and SHALL NOT panic or send a request.

#### Scenario: A zero-value TWSE/TPEx client
- **WHEN** 對 `var c TWStockClient` 與 `(*TWStockClient)(nil)` 呼叫六個抓取方法
- **THEN** 每一個都回傳指出要用 `TWStock` 建立的錯誤，不 panic

#### Scenario: A zero-value geocoding client
- **WHEN** 對 `var g TWGeocodingClient` 與 `(*TWGeocodingClient)(nil)` 呼叫 `Reverse`、`ReverseCols`、`ReverseTable`、`ReverseTableByColName`
- **THEN** 每一個都回傳指出要用 `TWGeocoding` 建立的錯誤，不 panic

#### Scenario: A zero-value Yahoo Finance client
- **WHEN** 對 `var y YFinanceClient` 取 `Ticker("AAPL")` 再呼叫 `Info`，以及對 `var t YFTicker` 呼叫 `Info`
- **THEN** 兩者都回傳指出要用 `YFinance` 建立的錯誤，不 panic

#### Scenario: A zero-value Google Maps client
- **WHEN** 對 `var c GoogleMapsStoresClient` 呼叫 `Search` 與 `GetReviews`
- **THEN** 兩者都回傳 nil 並記錄指出要用 `GoogleMapsStores` 建立的警告，不 panic
