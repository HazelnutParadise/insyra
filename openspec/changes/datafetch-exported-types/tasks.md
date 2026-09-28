# Tasks: datafetch-exported-types

## 1. Tests first

- [x] 1.1 新的外部測試檔（`package datafetch_test`）：以五個匯出型別宣告 struct 欄位，存入四個建構子與 `Ticker` 的回傳值（舊程式碼編譯失敗，即為紅）
- [x] 1.2 新測試：`TWStockClient`、`TWGeocodingClient`、`YFinanceClient`、`YFTicker` 的零值與 nil 指標呼叫每個抓取方法都回傳錯誤且不 panic；`GoogleMapsStoresClient` 回傳 nil 並記錄警告（舊程式碼 panic，即為紅）

## 2. Implementation

- [x] 2.1 `datafetch/`：`twStock`、`yahooFinance`、`ticker`、`twGeocoder`、`googleMapsStoreCrawler` 改名為 `TWStockClient`、`YFinanceClient`、`YFTicker`、`TWGeocodingClient`、`GoogleMapsStoresClient`，各加 doc comment；套件內測試跟著改名
- [x] 2.2 三個 HTTP 客戶端在公開方法入口檢查接收者是否由建構子建立；Yahoo Finance 的錯誤訊息改用匯出型別名稱並指出 `YFinance`

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/datafetch.md`：五個簽名改為匯出型別，說明零值不可用
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`datafetch`
- [x] 3.3 `api-review.md`：DF-3 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate datafetch-exported-types --strict`
