# Tasks: datafetch-context

## 1. Tests first

- [x] 1.1 TWStock（fixture transport）：六個 `Context` 方法在已取消的 context 下回傳 `context.Canceled` 且沒有請求；nil context 回傳錯誤；限流等待中、重試退避中到期都回傳 `DeadlineExceeded` 且只有一個請求；兩個月區間在第一個月後取消不再請求
- [x] 1.2 TWGeocoding：`ReverseContext` 已取消不請求；請求中被呼叫端期限切斷時回傳 `DeadlineExceeded` 而非 `ErrGeocodeTimeout` 且不重試；`ReverseColsContext` 中途取消保留已解析列、重複列、其餘 `pending`；`ReverseTableContext` 可用
- [x] 1.3 Google Maps：`SearchContext`、`GetReviewsContext` 已取消時回傳 nil 並記錄警告且不請求；兩頁之間等待中到期即回傳 nil，只有一個請求
- [x] 1.4 Yahoo Finance：26 個 `Context` 方法在已取消的 context 下回傳 `context.Canceled` 且不連網；限流等待中到期回傳 `DeadlineExceeded`；執行 go-yfinance 呼叫的輔助函式在呼叫不返回時依 context 立即回傳、呼叫 panic 時回傳錯誤、`context.Background()` 時在呼叫端的 goroutine 執行

## 2. Implementation

- [x] 2.1 共用：nil context 檢查、可被 context 中斷的等待
- [x] 2.2 `datafetch/twstock.go`：`doJSON` 與各列擷取函式帶 context，六個 `Context` 方法，退避可中斷
- [x] 2.3 `datafetch/geocoding.go`：`ReverseContext`、`ReverseColsContext`、`ReverseTableContext`，請求被 context 結束時回傳 `ctx.Err()`，批次中途取消的部分結果
- [x] 2.4 `datafetch/googleMapsCommentCrawler.go`：`SearchContext`、`GetReviewsContext`，頁間等待可中斷
- [x] 2.5 `datafetch/yfinance.go`：go-yfinance 呼叫輔助函式與建表輔助函式，26 個 `Context` 方法，`History`／`Quote` 的限流與退避帶 context

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/datafetch.md`：各客戶端的 `Context` 方法、取消時的回傳、Yahoo Finance 的限制
- [x] 3.2 `skills/insyra/SKILL.md`：`...Context` 形式的慣例涵蓋網路抓取
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：`datafetch`
- [x] 3.4 `AGENTS.md`：follow-up，只有 `History` 與 `Quote` 遵守 `YFinanceConfig.Interval` 與 `Retries`
- [x] 3.5 `api-review.md`：DF-2 標為已修正；`delivery-status.md`

## 4. Review

- [x] 4.1 對抗式審查四個 datafetch 改動，逐條核對並修正：補回改寫時被刪掉的 `Sustainability`；`HistoryContext` 在呼叫開始前轉換參數、`toModel` 複製日期，避免被放棄的呼叫與呼叫端修改競爭；被放棄的呼叫以 `runtime.KeepAlive` 保住 `YFinanceClient`，避免 finalizer 卡住；`runYF` 在 `context.Background()` 下於呼叫端 goroutine 執行、等待上限小於一秒時換成預設值，各補上能抓到退步的測試；文件、註解、changelog 改寫放棄呼叫的實際行為
- [x] 4.2 `AGENTS.md`：follow-up，`YFinanceConfig.Timeout` 小於一秒會變成 15 秒；被放棄的呼叫可能讓同一 client 的後續呼叫等待（尚未端到端驗證）

## 5. Verification

- [x] 5.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate datafetch-context --strict`
