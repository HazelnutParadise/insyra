# Tasks: sortby-empty-config-first-column

## 1. 實作
- [x] 1.1 沒有指定欄位的設定視為 `ColumnNumber: 0`，依第一欄排序，不報錯也不警告。
- [x] 1.2 CLI `sort` 以 `ColumnNumber` 傳數字位置，不再轉成 `ColumnIndex`。

## 2. 測試
- [x] 2.1 `{}`、只設 `Descending`、`{ColumnNumber: 0}` 依第一欄排序，`Err()` 為 nil，沒有警告。對 `sortby-column-selection` 的實作跑會失敗。
- [x] 2.2 只設 `ColumnName` 時依名稱排序，沒有警告。
- [x] 2.3 `TestDataTable_SortBy_ByIndex` 改回 `{ColumnNumber: 0}`。
- [x] 2.4 CLI `sort dt 0`、`sort dt <name>` 照常排序且沒有警告。

## 3. 收尾
- [x] 3.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 3.2 兩份 CHANGELOG 的 #233 條目改寫成相對上一版的實際變化，拿掉 BREAKING。
- [x] 3.3 `Docs/DataTable.md`、`api-review.md`、`delivery-status.md`，archive 後更新 `datatable-sort-config` 的 Purpose，在 #233 留言說明裁定。
