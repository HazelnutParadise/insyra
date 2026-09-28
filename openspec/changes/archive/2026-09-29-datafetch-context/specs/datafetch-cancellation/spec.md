## ADDED Requirements

### Requirement: Every fetch has a Context form

Each method below SHALL have a form named with `Context` appended that takes `ctx context.Context` as its first parameter and otherwise the same parameters and results, and the plain method SHALL behave as its `Context` form called with `context.Background()`:

- `TWStockClient`: `DailyPrices`, `DailyPricesAdjusted`, `ExRights`, `InstitutionalTrades`, `MarginBalance`, `AllDailyQuotes`.
- `TWGeocodingClient`: `Reverse`, `ReverseCols`, `ReverseTable`.
- `YFTicker`: `History`, `Quote`, `Info`, `Dividends`, `Splits`, `Actions`, `Options`, `OptionChain`, `News`, `Calendar`, `IncomeStatement`, `BalanceSheet`, `CashFlow`, `MajorHolders`, `InstitutionalHolders`, `MutualFundHolders`, `InsiderTransactions`, `FastInfo`, `EarningsEstimate`, `EarningsHistory`, `EPSTrend`, `EPSRevisions`, `Recommendations`, `AnalystPriceTargets`, `RevenueEstimate`, `GrowthEstimates`.
- `GoogleMapsStoresClient`: `Search`, `GetReviews`.

A `nil` context SHALL be refused with an error, or for the Google Maps client a `nil` result with a warning, and SHALL NOT panic or send a request.

#### Scenario: A context already cancelled
- **WHEN** 以已取消的 context 呼叫上列每個 `Context` 方法
- **THEN** TWSE/TPEx、地理編碼與 Yahoo Finance 的方法都回傳 `errors.Is(err, context.Canceled)` 成立的錯誤，Google Maps 的方法回傳 nil 並記錄警告，全部沒有發出請求

#### Scenario: A nil context
- **WHEN** 以 nil context 呼叫 `DailyPricesContext`、`ReverseContext`、`InfoContext`、`SearchContext`
- **THEN** 前三個回傳錯誤，`SearchContext` 回傳 nil 並記錄警告，沒有 panic，也沒有發出請求

### Requirement: The context reaches every wait and request

A `Context` method SHALL wait for the request limiter under `ctx`, SHALL stop a retry backoff when `ctx` is done, and for the TWSE/TPEx, geocoding and Google Maps clients SHALL send each HTTP request under `ctx`. When `ctx` is done during any of these, the call SHALL return `ctx.Err()` without starting another request and without retrying. A call that sends several requests (a month range, one-year ex-rights slices, the `TWMarketAuto` fallback, review pages) SHALL send none after `ctx` is done. `ReverseContext` SHALL report a request ended by `ctx` as `ctx.Err()`, not as `ErrGeocodeTimeout`.

#### Scenario: Cancelled while waiting for the limiter
- **WHEN** `Interval` 為 10 秒，第一次 `AllDailyQuotesContext` 成功後，第二次以 50 毫秒後到期的 context 呼叫
- **THEN** 第二次約 50 毫秒後回傳 `context.DeadlineExceeded`，只發出一個請求

#### Scenario: Cancelled during a retry backoff
- **WHEN** `Retries` 為 3、`RetryBackoff` 為 10 秒，第一個請求回應 HTTP 500，context 50 毫秒後到期
- **THEN** 約 50 毫秒後回傳 `context.DeadlineExceeded`，只發出一個請求

#### Scenario: Cancelled between months
- **WHEN** 以兩個月的區間呼叫 `DailyPricesContext`，第一個月的請求完成後取消 context
- **THEN** 回傳 `context.Canceled`，第二個月沒有發出請求

#### Scenario: A geocoding request cut off by the caller's deadline
- **WHEN** 伺服器延遲回應，`Retries` 為 2，context 50 毫秒後到期
- **THEN** `ReverseContext` 約 50 毫秒後回傳 `context.DeadlineExceeded` 而不是 `ErrGeocodeTimeout`，伺服器只收到一個請求

#### Scenario: Cancelled during the wait between review pages
- **WHEN** 第一頁回傳下一頁代碼，context 在兩頁之間的等待中到期
- **THEN** `GetReviewsContext` 在等待結束前回傳 nil 並記錄警告，只發出一個請求

### Requirement: A geocoding batch stopped by its context keeps what it resolved

When `ctx` is done part-way through `ReverseColsContext` or `ReverseTableContext`, the method SHALL return the table with every row resolved so far, the in-batch duplicates of those rows served from the batch's memo, every other row with a valid coordinate marked `pending`, and `ctx.Err()`, as it does for an exhausted quota. It SHALL NOT send another request after `ctx` is done.

#### Scenario: Cancelled after the first request
- **WHEN** 三個不同座標與一個重複第一個座標的列，第一個請求成功後取消 context
- **THEN** 第一列與重複列為 `ok`，其餘兩列為 `pending`，錯誤為 `context.Canceled`，伺服器只收到一個請求

### Requirement: A Yahoo Finance call in flight is abandoned, not interrupted

go-yfinance cannot stop a call once it has started, and one call can send several requests. When `ctx` is done while a Yahoo Finance call is running, the `YFTicker` method SHALL return `ctx.Err()` without waiting for the call, SHALL discard the call's result when it arrives, and SHALL NOT let a panic in that call end the program. The abandoned call SHALL share no memory with the caller's arguments, so a caller changing them afterwards cannot race it, and SHALL keep its `YFinanceClient` reachable until it ends, so the client's finalizer cannot run while the call still uses the client. With `context.Background()` the call SHALL run on the caller's goroutine as before.

#### Scenario: A call that does not return
- **WHEN** 被執行的 go-yfinance 呼叫一直不返回，而 context 50 毫秒後被取消
- **THEN** 約 50 毫秒後回傳 `context.Canceled`

#### Scenario: A panic in an abandoned call
- **WHEN** 在可取消的 context 下，被執行的呼叫 panic
- **THEN** 回傳錯誤，程式不中止

#### Scenario: A call without a cancel runs on the caller's goroutine
- **WHEN** 以 `context.Background()` 執行一個會 panic 的呼叫
- **THEN** panic 直接傳到呼叫端，而不是變成錯誤

#### Scenario: The history parameters are copied before the call
- **WHEN** 把帶 `Start`、`End` 與 `RepairOptions` 的 `YFHistoryParams` 轉成 go-yfinance 的參數後，呼叫端再修改原本的日期與修復選項
- **THEN** 轉換結果不受影響
