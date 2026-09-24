# Tasks: csvxl-respects-file-extension

## 1. 測試先紅
- [x] 1.1 輸入端：`export.txt`、`DATA.CSV`、無副檔名檔案可讀；`data`→`data.csv`、`export.txt`→`export.txt.csv` 補上後可讀；兩個都在時讀原路徑；同名資料夾時改讀補上的；兩個都不在時錯誤列出兩條路徑。
- [x] 1.2 輸出端：`report.txt`、`REPORT.CSV` 照用；`report` 補成 `report.csv`。

## 2. 實作
- [x] 2.1 `csvxl/convert.go` 輸入端與輸出端各一個小函式，兩處呼叫點改用。

## 3. 文件與紀錄
- [x] 3.1 `Docs/csvxl.md` 寫明兩條規則；兩份 CHANGELOG（輸出端標 BREAKING）。
- [x] 3.2 全套驗證；`api-review.md` C-8 與對照列、`delivery-status.md`；歸檔、寫 Purpose、關 #269。
