## ADDED Requirements

### Requirement: Sheet names cannot escape the output directory

`ExcelToCsv`／`EachExcelToCsv` SHALL 檢查每張工作表實際使用的 CSV 檔名（工作表名稱加 `.csv`、`EachExcelToCsv` 加上活頁簿檔名前綴的檔名，或呼叫端的 `csvNames` 項目）：檔名含 `/` 或 `\`，或與輸出目錄結合後不是輸出目錄內的檔案時 SHALL 回錯誤。每張工作表 SHALL 在寫入它自己的 CSV 之前檢查，且 SHALL 在建立 CSV 檔之前讀取完成。名為 `.` 或 `..` 的工作表 SHALL 照常轉換。

#### Scenario: Crafted sheet name
- **WHEN** workbook.xml 的工作表名為 `../important` 且 `outDir/../important.csv` 已存在
- **THEN** 回傳含「sheet name」的錯誤，該檔內容不變

#### Scenario: A sheet named with dots
- **WHEN** 工作表名為 `..`
- **THEN** 寫出 `outDir/...csv`，不回錯誤

#### Scenario: A csvNames entry that leaves the output directory
- **WHEN** 呼叫端以 `csvNames` 指定 `../escape`
- **THEN** 回傳錯誤，輸出目錄外不會寫出任何檔案
