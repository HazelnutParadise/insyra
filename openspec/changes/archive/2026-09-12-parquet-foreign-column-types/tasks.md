# Tasks: parquet-foreign-column-types

## 1. 先重現
- [x] 1.1 寫一份含九種外來欄位型別的 parquet 檔，用 `Read` 讀回並斷言值正確，確認修正前是紅的。

## 2. 依賴
- [x] 2.1 加入 `github.com/TimLai666/go-decimal`，確認 `go` 指令不被抬高。
- [x] 2.2 確認沒有其他子模組需要同步（`accel/backend/wgpu` 早已併回核心模組，倉庫只有一個 go.mod）。

## 3. 讀取路徑
- [x] 3.1 `getVal` 補上日期、整數寬度、二進位、十進位的分支。
- [x] 3.2 `getVal` 的 `default` 回傳 `nil`，不再回傳整欄字串。
- [x] 3.3 `chunkedToSlice` 的 typed 快速路徑與 `supportedArrowType` 用同一份型別清單，避免兩邊漂移。
- [x] 3.4 欄位層級偵測不支援的型別，在 DataTable 上記錄欄名與 Arrow 型別。
- [x] 3.5 `Read`、`Stream`（`recordToDataTable`）、`ReadColumn` 三個入口都要記錄。

## 4. 十進位的排序
- [x] 4.1 `GetTypeSortingRank` 與 `CompareAny` 比照 `time.Time` 加入十進位分支。
- [x] 4.2 `engine/algorithms` 的再匯出確認不受影響。

## 5. 測試
- [x] 5.1 每個新支援的型別逐值斷言，含 38 位數 Decimal128 與負數。
- [x] 5.2 不支援的型別為 nil 且 `Err()` 指出欄名與型別。
- [x] 5.3 全部型別都支援時 `Err()` 為 nil。
- [x] 5.4 `supportedArrowType` 與 `getVal` 的清單一致（防漂移）。
- [x] 5.5 十進位排序。
- [x] 5.6 `FilterWithCCL` 在含不支援欄位的檔案上不會拿到假值。

## 6. 收尾
- [x] 6.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 6.2 `Docs/parquet.md` 的型別對照表。
- [x] 6.3 兩份 CHANGELOG。
- [x] 6.4 `skills/insyra/` 若有提到 parquet 型別就更新。
- [x] 6.5 `api-review.md`、`delivery-status.md`、`AGENTS.md`（移除 #371 的 Follow-up，新增寫回仍失真與十進位非數值兩項）。
- [x] 6.6 關閉 #371 並附證據。
