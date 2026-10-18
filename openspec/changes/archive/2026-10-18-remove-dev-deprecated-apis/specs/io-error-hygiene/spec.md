## MODIFIED Requirements

### Requirement: csvxl errors unwrap and directories are 0755

`csvxl` 的錯誤 SHALL 以 `%w` 包裝底層錯誤；`ExcelToCSV`／`ExcelDirToCSV` 建立的目錄權限 SHALL 為 0755。

#### Scenario: Missing CSV
- **WHEN** `CSVToExcel([]SheetFromCSV{{Path: "/nope.csv"}}, out)`
- **THEN** 回傳的錯誤 `errors.Is(err, os.ErrNotExist)` 為 true

### Requirement: A named sheet that is not there is an error

`ExcelToCSV` 收到 `Sheets` 時，其中不存在於檔案的名稱 SHALL 回報錯誤並列出檔案實際有的工作表，SHALL NOT 靜默略過。

#### Scenario: A misspelled sheet name
- **WHEN** `Sheets` 含一個檔案裡沒有的名稱
- **THEN** 回傳錯誤，指出缺少哪些、檔案有哪些
