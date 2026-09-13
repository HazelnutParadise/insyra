# Tasks: show-a-cell-that-knows-its-own-text

## 1. 先重現
- [x] 1.1 測試：一個只有 `String()` 的 struct 值，`FormatValue` 回 `<型別名>`。
- [x] 1.2 測試：同樣的值經過 `ToJSON_String` 變成 `{}`。
- [x] 1.3 測試：`time.Time` 的 JSON 維持 RFC 3339（修正前後都要綠，防止退步）。

## 2. 實作
- [x] 2.1 `FormatValue` 在 struct 分支之前先問 `fmt.Stringer`。
- [x] 2.2 `buildJSONRows` 對「有 `String()` 但沒有 `json.Marshaler`／`encoding.TextMarshaler`」的值改寫成文字。

## 3. 收尾
- [x] 3.1 `gofmt -l .`、`go build ./...`、`go test ./...`、`golangci-lint run`。
- [x] 3.2 以實際讀進來的 parquet 十進位欄位確認 `Show()` 與 `ToJSON` 的輸出。
- [x] 3.3 兩份 CHANGELOG。
- [x] 3.4 修正 `Docs/parquet.md` 與已封存的 `parquet-foreign-column-types` proposal 裡「numeric path 會指名列號拒絕」的錯誤說法，並把 `Mean` 回 NaN 的實測補進 `AGENTS.md` 那條 follow-up。
