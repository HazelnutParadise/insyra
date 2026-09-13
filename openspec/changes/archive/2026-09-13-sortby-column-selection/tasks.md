# Tasks: sortby-column-selection

## 1. 先重現
- [x] 1.1 測試：空設定、只設 `Descending`、單獨 `{ColumnNumber: 0}` 目前默默依第一欄排序。
- [x] 1.2 測試：`ColumnIndex` 找不到欄時表格不變且 `Err()` 為 nil；名稱與數字找不到時錯誤指向內部函式。
- [x] 1.3 測試：多層排序中一層無效，其餘層仍被套用。

## 2. 實作
- [x] 2.1 先解析全部設定再移動任何一列；任一無效就以 `SortBy` 為名 `fail` 並返回。
- [x] 2.2 以不記錄錯誤的內部查找解析欄位，數字直接檢查範圍。
- [x] 2.3 多個欄位同時指定時 `warn`，依索引、名稱、數字的順序選欄。
- [x] 2.4 CLI `sort` 只設它解析出的那個欄位，數字轉成 `ColumnIndex`。

## 3. 測試
- [x] 3.1 空設定、只設 `Descending`、單獨 `{ColumnNumber: 0}` 被拒絕且表格不變；`ColumnIndex: "A"` 可用。
- [x] 3.2 三種找不到的情況錯誤都指名 `SortBy`。
- [x] 3.3 多層排序全有全無。
- [x] 3.4 索引加名稱、名稱加非零數字：依優先順序排序、有警告、`Err()` 為 nil。
- [x] 3.5 既有以 `{ColumnNumber: 0}` 排序的測試改用 `ColumnIndex: "A"`（行為刻意改變）。
- [x] 3.6 CLI `sort t 0` 與 `sort t <name>` 都能排序且不出警告。

## 4. 收尾
- [x] 4.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 4.2 兩份 CHANGELOG（BREAKING）。
- [x] 4.3 `Docs/DataTable.md` 寫明優先順序、零值規則與錯誤。
- [x] 4.4 `api-review.md` T-23、`delivery-status.md`；在 #225 記錄 fallback 的裁定；關閉 #233。
