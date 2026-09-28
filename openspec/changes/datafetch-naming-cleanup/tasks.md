# Tasks: datafetch-naming-cleanup

## 1. Tests first

- [x] 1.1 新測試：解析 doc comment，`YFPeriodYearly` 與四個舊排序常數都有指名替代名稱的 `Deprecated:` 段落，舊常數與新常數數值相同，`YFPeriodYearly` 仍為 `"yearly"`（舊程式碼缺新常數，即為紅）
- [x] 1.2 新測試：`MaxWaitingInterval: time.Second` 抓兩頁至少等一秒且無警告；兩個等待欄位都設定時回傳 nil、有警告、沒有請求；小於一秒時有警告仍取得頁面
- [x] 1.3 新測試：`ReverseTable` 以字母、`insyra.Name`、整數三種寫法結果相同；`"lat"` 這種裸字串回傳錯誤且不發請求；`ReverseTableByColName` 的 doc comment 有 `Deprecated:` 段落，結果與 `Name` 寫法相同

## 2. Implementation

- [x] 2.1 `datafetch/yfinance.go`：`YFPeriodYearly` 加 Deprecated
- [x] 2.2 `datafetch/googleMapsCommentCrawler.go`：四個新排序常數，舊名改為 Deprecated 常數；`MaxWaitingInterval time.Duration` 欄位與兩欄衝突的處理，等待改以 `time.Duration` 計算；舊欄位加 Deprecated
- [x] 2.3 `datafetch/geocoding.go`：`ReverseTable(dt, latCol, lngCol any)`，錯誤訊息指出哪一欄的選擇器；`ReverseTableByColName` 改為 Deprecated 並呼叫 `ReverseTable`

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/datafetch.md`：排序常數、等待欄位、`ReverseTable` 的選擇器、`YFPeriodYearly`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`datafetch`
- [x] 3.3 `AGENTS.md`：下一版移除 Deprecated 名稱的 follow-up
- [x] 3.4 `api-review.md`：DF-5 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate datafetch-naming-cleanup --strict`
