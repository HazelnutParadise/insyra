# Tasks: filter-rows-where

## 1. Tests first

- [x] 1.1 `datatable_filter_rows_where_test.go` 與 `datatable_slice_review_test.go`：`FilterRowsWhere` 比較兩欄、依欄位順序傳整列、名稱與 cell 保留、複製獨立性、全不過保留欄名、nil predicate 回報錯誤；predicate 增刪欄位不崩潰。
- [x] 1.2 `Filter` 任一格通過即保留列並傳遞欄位字母；`FilterByCustomElement` 等於 `Filter`。

## 2. Implementation

- [x] 2.1 `datatable_filters.go`：`FilterRowsWhere`；`Filter`、`FilterRows` 與 `FilterRowsWhere` 使用呼叫開始時的欄位切片複本；`FilterByCustomElement` 標為 Deprecated 並註明在 0.4 移除。
- [x] 2.2 `interfaces.go`：`IDataTable` 列出 `FilterRowsWhere`。

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`：`Filter` 與 `FilterRows` 說明為任意格測試；更正範例；`FilterByCustomElement` 標記 Deprecated；新增 `FilterRowsWhere` 章節。
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core。
- [x] 3.3 `api-review.md`、`delivery-status.md`。

## 4. Verification

- [x] 4.1 全套驗證。
