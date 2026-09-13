## ADDED Requirements

### Requirement: Sheet names cannot escape the output directory

`ExcelToCsv`／`EachExcelToCsv` 對含 `/`、`\`、為 `..` 或 `filepath.Base` 不等於自身的工作表名稱 SHALL 回錯誤，且 SHALL 在碰任何輸出檔之前檢查；CSV SHALL 在工作表讀取成功後經暫存檔寫入。

#### Scenario: Crafted sheet name
- **WHEN** workbook.xml 的工作表名為 `../important` 且 `outDir/../important.csv` 已存在
- **THEN** 回傳含「sheet name」的錯誤，該檔內容不變
