# csvxl-sheet-name-safety Specification

## Purpose
`ExcelToCsv`／`EachExcelToCsv` 的工作表名稱安全契約：名稱不得逃出輸出目錄，讀不出來的工作表也不得截斷既有的 CSV 檔。

## Requirements
### Requirement: Sheet names cannot escape the output directory

`ExcelToCsv`／`EachExcelToCsv` 對含 `/`、`\`、為 `..` 或 `filepath.Base` 不等於自身的工作表名稱 SHALL 回錯誤，且 SHALL 在碰任何輸出檔之前檢查；工作表 SHALL 在建立 CSV 檔之前讀取完成。

#### Scenario: Crafted sheet name
- **WHEN** workbook.xml 的工作表名為 `../important` 且 `outDir/../important.csv` 已存在
- **THEN** 回傳含「sheet name」的錯誤，該檔內容不變
