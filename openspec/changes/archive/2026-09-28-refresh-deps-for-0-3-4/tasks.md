# Tasks: refresh-deps-for-0-3-4

- [x] 1.1 `go get -u -t ./... go@1.25.12`；`go mod tidy`
- [x] 1.2 確認 `go.mod` 的 `go` 指令仍是 1.25.12，沒有新增 `toolchain` 行
- [x] 1.3 `chromedp` 拉回 v0.12.1，`cdproto` 用它要求的版本，`go-json-experiment/json` 不在依賴圖裡
- [x] 1.4 `goccy/go-json` 留在 v0.10.6：新舊版對 `ToJSON`、`ReadJSON` 與解碼進結構的差異實測記錄在 `AGENTS.md`
- [x] 1.5 `google.golang.org/grpc` v1.84.0：GitHub advisory 的範圍外，並用 compare API 確認 v1.84.0 含修正 commit `d5a41119`
- [x] 2.1 `go build ./...`、`go vet ./...`、`go test ./...` 全綠，`golangci-lint run` 0 issues
- [x] 2.2 `INSYRA_ACCEL_GPU_TESTS=1 go test ./accel/... ./nn/...` 在 Apple M3（Metal）上全綠
- [x] 2.3 CI 的 govulncheck v1.3.0 用 Go 1.25.14 跑完不當掉：`Your code is affected by 0 vulnerabilities`，只在「有 import 沒呼叫」與「只在 module」兩層列出 GO-2026-6443（grpc，Go 資料庫範圍過期）和 #203 追蹤的 x/crypto 三則
- [x] 2.4 GitHub advisory database 逐一檢查這次移動的 8 個模組，全部在每個範圍外
- [x] 2.5 `datafetch.YFinance` 對 AAPL 的 History（2024-01 固定區間）、Dividends、Quote 欄位，新舊依賴輸出一致；調整後收盤價的末位差異在舊依賴連跑三次時也出現，是 Yahoo 端的變動
- [x] 2.6 Parquet 以 uncompressed、snappy、gzip、brotli、zstd 寫入並交叉讀取，新舊依賴寫出的檔案逐位元相同，讀出的內容一致
- [x] 3.1 `AGENTS.md`：刷新規則加上「改變結果的升級另開變更」；Follow-ups 更新被 Go 1.25 擋住的模組，新增 go-json 一條
- [x] 3.2 `openspec validate refresh-deps-for-0-3-4 --strict` 通過
