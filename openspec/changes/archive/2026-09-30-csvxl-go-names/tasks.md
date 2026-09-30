# Tasks: csvxl-go-names

## 1. Tests first

- [x] 1.1 新測試：解析非測試原始檔，每個匯出識別字的 doc comment 以其名稱開頭、套件註解以 `Package csvxl` 開頭；七個舊名稱有指名新名稱的 `Deprecated:` 段落（舊程式碼缺新名稱與 Go 風格註解，即為紅）
- [x] 1.2 新測試：`CSVToExcel`、`AppendCSVToExcel`、`CSVDirToExcel`、`ReadCSVToString` 對 `"klingon-1"` 回傳含該名稱的錯誤且不寫出工作簿；`CsvToExcel` 與 `CSVToExcel` 結果相同；`CSVDirToExcel` 一個 CSV 一張工作表
- [x] 1.3 既有測試改用新名稱，另留測試確認舊名稱的結果不變
- [x] 1.4 新測試：沒有檔案時不存在的編碼回傳錯誤且不寫工作簿；三個檔案時錯誤只提一次編碼名稱；已知編碼在沒有檔案時照舊寫出工作簿（前兩項在舊程式碼上為紅）

## 2. Implementation

- [x] 2.1 `csvxl/convert.go`、`csvxl/convertDir.go`、`csvxl/read_csv.go`：新名稱、Deprecated 包裝與別名、Go 風格 doc comment、log 的函式名稱
- [x] 2.2 `csvxl/init.go`：套件註解以 `Package csvxl` 開頭，範例改用新名稱
- [x] 2.3 `cli/commands/convert.go`：改呼叫新名稱
- [x] 2.4 `csvxl/convert.go`：`CSVToExcel`、`AppendCSVToExcel` 在讀任何檔案之前先檢查編碼名稱（審查發現空清單時不會檢查，多檔時錯誤重複）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/csvxl.md`：新名稱、舊名稱的對照
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`csvxl`
- [x] 3.3 `AGENTS.md`：下一版移除舊名稱的 follow-up
- [x] 3.4 `api-review.md`：C-6、C-12 標為已修正；`delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate csvxl-go-names --strict`
