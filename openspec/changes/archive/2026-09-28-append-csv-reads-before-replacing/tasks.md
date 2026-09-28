# Tasks: append-csv-reads-before-replacing

## 1. Implementation

- [x] 1.1 `csvxl/convert.go`：把讀取 CSV（開檔、偵測編碼、解碼、`ReadAll`、去 BOM）與寫入儲存格拆開；`addCsvSheet` 仍是兩者依序呼叫，`CsvToExcel` 行為不變
- [x] 1.2 `AppendCsvToExcel` 先讀完 CSV，成功才 `replaceSheet` 再寫入；讀取失敗算一個失敗檔案並跳過，錯誤文字不變

## 2. Tests

- [x] 2.1 測試涵蓋 spec 的四個情境：CSV 不存在、CSV 內容無效、同批一好一壞、目標工作表原本不存在
- [x] 2.2 三個測試在修正前的程式上失敗（工作表被清空），修正後通過
- [x] 2.3 `go test ./csvxl/...`、`go vet ./...`、`golangci-lint run` 全綠

## 3. Records

- [x] 3.1 `Docs/csvxl.md` 的 `AppendCsvToExcel` 說明讀不到的 CSV 不會動到它的工作表
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` → `### csvxl` 各加一條
- [x] 3.3 刪除 `AGENTS.md` Follow-ups 中「`AppendCsvToExcel` empties a sheet before it knows the CSV can be read」
- [x] 3.4 `openspec validate append-csv-reads-before-replacing --strict` 通過
