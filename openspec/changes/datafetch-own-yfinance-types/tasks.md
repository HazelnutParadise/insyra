# Tasks: datafetch-own-yfinance-types

## 1. Tests first

- [x] 1.1 新測試：解析 `datafetch` 非測試檔，匯出宣告不得引用第三方模組的型別（舊程式碼因 `YFHistoryParams` 與 `News` 而紅）
- [x] 1.2 新測試：`YFHistoryParams`／`YFRepairOptions` 與 go-yfinance 對應結構的欄位名稱集合相同；每個欄位設非零值後轉換結果逐欄相同；nil `RepairOptions` 轉成 nil
- [x] 1.3 新測試：`News` 收到未知分頁時回傳錯誤且不發請求；空分頁轉成 news

## 2. Implementation

- [x] 2.1 `datafetch/yfinance.go`：`YFHistoryParams`、`YFRepairOptions` 改為自有結構（欄位與 JSON tag 同 go-yfinance），加內部轉換函式，`History` 改用轉換結果
- [x] 2.2 `YFNewsTab` 與三個常數，`News(count int, tab YFNewsTab)`，未知分頁在發請求前回傳錯誤

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/datafetch.md`：`YFHistoryParams` 全部欄位、`YFRepairOptions`、`News` 與 `YFNewsTab`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`datafetch`，標 BREAKING
- [x] 3.3 `api-review.md`：DF-4 的型別部分標為已修正，User-Agent 部分留待擁有者；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate datafetch-own-yfinance-types --strict`
