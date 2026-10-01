# Tasks: parquet-ccl-whole-file-aggregates

## 1. internal/ccl

- [x] 1.1 `GlobalRowContext`：`#` 經 `GlobalRowIndex` 取值；測試其他 context 不受影響（先寫失敗測試）
- [x] 1.2 `NewStreamingAggregate`：九個內建彙總的串流版本，任意分批餵入的結果與彙總函式一次給全部值時逐位元相同；被呼叫端重新註冊的名稱沒有串流版本（先寫失敗測試）
- [x] 1.3 `ResolveWholeTable`：固定列與列範圍的擷取、彙總依巢狀層級分批計算並共用讀檔次數、不支援的部分回傳指出它的錯誤；以記憶體中的分批來源測試結果與整表計算相同（先寫失敗測試）

## 2. parquet

- [x] 2.1 `parquetContext` 帶批次在檔案中的位置並實作 `GlobalRowContext`；`FilterWithCCL` 與 `ApplyCCL` 先解出整檔部分（`ApplyCCL` 逐句）再逐批計算；以載入成表格的結果為基準比對（先寫失敗測試）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/parquet.md`：哪些部分改成整檔計算、要多讀幾次、哪些暫時拒絕
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`parquet`，標 BREAKING（原本給出逐批答案的寫法改為報錯）
- [x] 3.3 `AGENTS.md`：follow-up 縮小為後兩個 change 的範圍；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate parquet-ccl-whole-file-aggregates --strict`
