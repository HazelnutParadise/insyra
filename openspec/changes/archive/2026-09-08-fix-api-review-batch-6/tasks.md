# Tasks: fix-api-review-batch-6

## 1. Tests first

- [x] 1.1 `batch6_test.go`：ISO-8859-1 檔讀出有效 UTF-8、UTF-32／16 BOM 判定、偵測失敗退回 utf-8、`SanitizeFormulas` 開關兩種輸出
- [x] 1.2 `datatable_to_sql_batch6_test.go`：含空白與含引號的資料表名稱可建立並附加

## 2. Implementation

- [x] 2.1 `internal/csv/decoder.go`：`DecodingReader`，未知編碼回錯誤
- [x] 2.2 `internal/csv/read_csv.go`、`csvxl/convert.go` 改用它
- [x] 2.3 `utils.go`：UTF-32 BOM 先判、偵測失敗退回 utf-8 並警告
- [x] 2.4 `datatable_csv.go`：`CSVWriteOptions`、`ToCSVWithOptions`、`sanitizeCSVFormula`
- [x] 2.5 `datatable_to_sql.go`：PRAGMA 的識別字加引號

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/DataTable.md`（`ToCSVWithOptions` 與公式注入）、`Docs/csvxl.md`（支援的編碼）
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core、`csvxl`
- [x] 3.3 `api-review.md`：SEC-3、SEC-4、SEC-5 標已修正；關閉 #285、#286、#287

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate fix-api-review-batch-6 --strict` 通過
