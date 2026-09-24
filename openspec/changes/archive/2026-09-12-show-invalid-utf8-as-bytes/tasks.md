# Tasks: show-invalid-utf8-as-bytes

## 1. 先重現
- [x] 1.1 測試：非法 UTF-8 的字串目前被加引號輸出，而且結果本身不是合法 UTF-8。
- [x] 1.2 測試：合法 UTF-8（含中日韓與 emoji）與多行字串的既有行為，修正前後都要綠。

## 2. 實作
- [x] 2.1 `FormatValue` 的 `string` 分支在最前面檢查 `utf8.ValidString`，不合法就走 `[]byte` 的顯示方式。

## 3. 收尾
- [x] 3.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 3.2 以實際的 parquet Binary 欄位確認顯示與欄位對齊。
- [x] 3.3 兩份 CHANGELOG；`Docs/parquet.md` 的 Binary 那一列補上顯示方式。
