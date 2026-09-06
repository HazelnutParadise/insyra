## ADDED Requirements

### Requirement: csvxl errors unwrap and directories are 0755

`csvxl` 的錯誤 SHALL 以 `%w` 包裝底層錯誤；`ExcelToCsv`／`EachExcelToCsv` 建立的目錄權限 SHALL 為 0755。

#### Scenario: Missing CSV
- **WHEN** `CsvToExcel([]string{"/nope.csv"}, nil, out)` 
- **THEN** 回傳的錯誤 `errors.Is(err, os.ErrNotExist)` 為 true

### Requirement: parquet.Write is atomic and logs through Insyra

`parquet.Write` SHALL 先寫暫存檔再 rename；`parquet` 套件 SHALL NOT 使用標準 `log` 套件輸出。

#### Scenario: Successful write leaves no temp file
- **WHEN** `Write(dt, path)` 成功
- **THEN** 目錄中只有 `path`
