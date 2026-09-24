# Tasks: reader-writer-entry-points

## 1. CSV
- [ ] 1.1 測試先紅：`ReadCSV` 與檔案版結果相同、Big5 在 reader 上自動偵測、`WriteCSV` 與檔案版位元組相同。
- [ ] 1.2 編碼偵測改成吃樣本，`DetectEncoding` 與 reader 版共用；`ReadCSV`、`WriteCSV`，路徑版與字串版改成包裝。
- [ ] 1.3 `StreamCSV`：測試先紅（10、10、5 列、每批都有欄名、中途 break），再實作。

## 2. JSON 與 Excel
- [ ] 2.1 `ReadJSON` 收 `io.Reader`；`WriteJSON`，`ToJSON` 包裝它。
- [ ] 2.2 `ReadExcel(r, …)`，套用相同的解壓上限；`ReadExcelSheet` 包裝它。

## 3. Parquet
- [ ] 3.1 `ReadFrom`、`StreamFrom`、`WriteTo`；`Read`、`Stream`、`Write` 包裝它們；測試用記憶體中的位元組讀寫。

## 4. 文件與紀錄
- [ ] 4.1 `Docs/DataTable.md`、`Docs/parquet.md`、skills；兩份 CHANGELOG。
- [ ] 4.2 全套驗證；`api-review.md` K-11、C-10、Q-8、SEC-16；`delivery-status.md`；歸檔、寫 Purpose；關 #210、#294。
