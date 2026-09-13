# Tasks: parallel-runs-and-worker-errors

## 1. 測試（先寫、先看它失敗）
- [x] 1.1 既有行為改寫到新 API：多種回傳型別、沒有回傳值、`AwaitNoResult` 會等待。
- [x] 1.2 group 跑兩次各自有結果；多個 goroutine 同時 Run（`-race`）；`GroupUp` 複製參數 slice；等待同一次執行兩次結果相同；空 group。
- [x] 1.3 函式自己回傳的 error 留在結果格；panic 成為 `*WorkerError`（Index、Panic、Stack）；`panic(err)` 可用 `errors.Is`；無法呼叫的值（非函式、nil、nil 函式、需要參數）；多個失敗依序回報；`AwaitNoResult` 回傳同一個錯誤。
- [x] 1.4 `*ParallelGroup` 沒有 Await 方法、`*RunningGroup` 沒有 Run 方法。

## 2. 實作
- [x] 2.1 `ParallelGroup`、`RunningGroup`、`WorkerError`，`Run` 每次建立獨立的結果。
- [x] 2.2 `mkt.RFM` 與 `stats.RepeatedMeasuresANOVA` 檢查 `AwaitResult` 的錯誤。

## 3. 文件與紀錄
- [x] 3.1 重寫 `Docs/parallel.md`；更新 `Docs/tutorials/python-enrichment-and-parallel-batch.md`；修正 `AGENTS.md` 的 parallel 說明。
- [x] 3.2 兩份 CHANGELOG 新增 `parallel` 段落的 BREAKING 條目。
- [x] 3.3 `api-review.md` P-1、P-2、P-4、P-5 與 parallel 符號清單，`delivery-status.md`。
- [x] 3.4 `gofmt -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`go test -race ./parallel/ ./mkt/ ./stats/`、`golangci-lint run`。
- [ ] 3.5 歸檔並寫 `parallel-groups` 的 Purpose；在 #271 留言附證據並關閉。
