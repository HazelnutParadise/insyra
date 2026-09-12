# Tasks: parquet-filter-keeps-every-batch

## 1. Reproduce first

- [x] 1.1 在 `parquet/ccl_test.go` 加一個跨批次的過濾測試（2500 列、條件全成立），確認它在修正前是紅的。
- [x] 1.2 再加一個「符合的列分散在不同批次」的測試，斷言值與順序。

## 2. Fix

- [x] 2.1 `FilterWithCCL` 改成整段串流收集到區域切片，結束後一次建表；刪掉第一批／後續批的分支。
- [x] 2.2 紀錄通道關閉後，`FilterWithCCL` 與 `ApplyCCL` 先讀錯誤通道再回傳。
- [x] 2.3 `Stream` 在關閉自己的輸出前做同樣的事。
- [x] 2.4 `streamAsArrowRecord` 關檔時忽略 `os.ErrClosed`。

## 3. Verify

- [x] 3.1 1.1 與 1.2 轉綠，`parquet` 其餘測試不變。
- [x] 3.2 成功的 `FilterWithCCL` 不再印出關檔警告（測試輸出裡沒有那行）。
- [x] 3.3 `go test -race ./parquet/`、`go test ./...`、`golangci-lint run`。

## 4. Ledger

- [x] 4.1 兩份 CHANGELOG 的 `## Unreleased` 加 `### parquet` 條目。
- [x] 4.2 `Docs/parquet.md` 說明 `FilterWithCCL` 讀取失敗時回傳錯誤。
- [x] 4.3 `delivery-status.md` 里程碑。
