# Tasks: reader-writer-entry-points

## 1. CSV
- [x] 1.1 測試先紅：`ReadCSV` 與檔案版結果相同、Big5 在 reader 上自動偵測、`WriteCSV` 與檔案版位元組相同。
- [x] 1.2 編碼偵測改成吃樣本，`DetectEncoding` 與 reader 版共用；`ReadCSV`、`WriteCSV`，路徑版與字串版改成包裝。
- [x] 1.3 `StreamCSV`：測試先紅（10、10、5 列、每批都有欄名、中途 break），再實作。

## 2. JSON 與 Excel
- [x] 2.1 `ReadJSON` 收 `io.Reader`；`WriteJSON`，`ToJSON` 包裝它。
- [x] 2.2 `ReadExcel(r, …)`，套用相同的解壓上限；`ReadExcelSheet` 包裝它。

## 3. 文件與紀錄
- [x] 3.1 `Docs/DataTable.md`；兩份 CHANGELOG。
- [x] 3.2 驗證與回報。
