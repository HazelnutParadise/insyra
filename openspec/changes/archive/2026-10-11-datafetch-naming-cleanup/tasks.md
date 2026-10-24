# Tasks: datafetch-naming-cleanup

## 1. Tests first

- [x] 1.1 新測試：解析 doc comment，`YFPeriodYearly` 與四個舊排序常數都有指名替代名稱的 `Deprecated:` 段落，舊常數與新常數數值相同，`YFPeriodYearly` 仍為 `"yearly"`（舊程式碼缺新常數，即為紅）
- [x] 1.2 新測試：`MaxWaitingInterval: time.Second` 抓兩頁至少等一秒且無警告；兩個等待欄位都設定時回傳 nil、有警告、沒有請求；小於一秒時有警告仍取得頁面

## 2. Implementation

- [x] 2.1 `datafetch/yfinance.go`：`YFPeriodYearly` 加 Deprecated
- [x] 2.2 `datafetch/googleMapsCommentCrawler.go`：四個新排序常數，舊名改為 Deprecated 常數；`MaxWaitingInterval time.Duration` 欄位與兩欄衝突的處理，等待改以 `time.Duration` 計算；舊欄位加 Deprecated

## 3. Docs and changelog

- [x] 3.1 `Docs/datafetch.md`：排序常數、等待欄位、`YFPeriodYearly`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`datafetch`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`