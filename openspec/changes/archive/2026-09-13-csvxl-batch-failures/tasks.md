# Tasks: csvxl-batch-failures

## 1. 測試（先寫、先看它失敗）
- [x] 1.1 `CsvToExcel` 批次中一個檔案不存在：輸出只有成功的工作表，錯誤指名該檔且 `errors.Is(err, os.ErrNotExist)`。
- [x] 1.2 兩個檔案失敗、其中一個排第一：錯誤列出兩個路徑，輸出不留預設的 `Sheet1`。
- [x] 1.3 工作表名稱含 `:`：該檔以錯誤回報，其他檔案照常寫入。
- [x] 1.4 全部失敗：`CsvToExcel` 不建立輸出檔；`AppendCsvToExcel` 不改寫工作簿。
- [x] 1.5 `AppendCsvToExcel` 讀不到某個 CSV：既有同名工作表不變，其他 CSV 照常取代。
- [x] 1.6 `EachCsvToOneExcel` 目錄裡有讀不了的 `.csv`：錯誤指名它，其他檔案照常寫入。

## 2. 實作
- [x] 2.1 把讀取 CSV 和寫入工作表拆開，先讀完再建立或取代工作表。
- [x] 2.2 `CsvToExcel`、`AppendCsvToExcel` 收集每個檔案的錯誤並以 `errors.Join` 回傳，至少一個成功才存檔。

## 3. 文件與紀錄
- [x] 3.1 `Docs/csvxl.md` 寫明三個函式遇到失敗檔案的行為；`skills/insyra/SKILL.md` 的 csvxl 範例改為檢查錯誤。
- [x] 3.2 兩份 CHANGELOG 的 `csvxl` BREAKING 條目。
- [x] 3.3 `api-review.md` C-1，`delivery-status.md`。
- [x] 3.4 `gofmt -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`。
- [x] 3.5 歸檔並寫 `csvxl-batch-conversion` 的 Purpose；在 #267 留言附證據並關閉。
