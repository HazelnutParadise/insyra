# csvxl-sheet-name-safety Specification

## Purpose
`ExcelToCsv` 與 `EachExcelToCsv` 寫出的 CSV 檔名，不論來自工作簿的工作表名稱還是呼叫端的 `csvNames`，都不得逃出輸出目錄。
## Requirements
### Requirement: Sheet names cannot escape the output directory

`ExcelToCSV`／`ExcelDirToCSV`（以及已棄用的 `ExcelToCsv`／`EachExcelToCsv`）SHALL 檢查每張工作表實際使用的 CSV 檔名（工作表名稱加 `.csv`、`ExcelDirToCSV` 加上活頁簿檔名前綴的檔名，或呼叫端在 `FileNames`／`csvNames` 指定的名稱）：檔名含 `/` 或 `\`，或與輸出目錄結合後不是輸出目錄內的檔案時 SHALL 回錯誤。所有工作表的檔名 SHALL 在寫入第一個 CSV 之前檢查完畢，所以被拒絕的檔名不會留下部分輸出；CSV SHALL 在工作表讀取成功後經暫存檔寫入。

#### Scenario: Crafted sheet name
- **WHEN** workbook.xml 的工作表名為 `../important` 且 `outDir/../important.csv` 已存在
- **THEN** 回傳含「sheet name」的錯誤，該檔內容不變

#### Scenario: A sheet named with dots
- **WHEN** 工作表名為 `..`
- **THEN** 寫出 `outDir/...csv`，不回錯誤

#### Scenario: A FileNames entry that leaves the output directory
- **WHEN** 呼叫端以 `FileNames` 把某張工作表指定為 `../escape`
- **THEN** 回傳錯誤，輸出目錄內外都不會寫出任何 CSV

#### Scenario: A csvNames entry that leaves the output directory
- **WHEN** 呼叫端以已棄用 `ExcelToCsv` 的 `csvNames` 指定 `../escape`
- **THEN** 回傳錯誤，輸出目錄外不會寫出任何檔案

