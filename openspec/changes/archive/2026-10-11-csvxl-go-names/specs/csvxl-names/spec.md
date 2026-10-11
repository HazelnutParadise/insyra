## ADDED Requirements

### Requirement: csvxl exports the CSV-spelled names

`csvxl` SHALL export `CSVToExcel`, `AppendCSVToExcel`, `CSVDirToExcel` and `ReadCSVToString`, each taking the arguments and doing what its old spelling does.

#### Scenario: A directory of CSV files
- **WHEN** 目錄裡有兩個 CSV，呼叫 `CSVDirToExcel(dir, out)`
- **THEN** `out` 是一個工作簿，每個 CSV 各一張工作表

### Requirement: The new names have doc comments that start with the name

Each of `CSVToExcel`, `AppendCSVToExcel`, `CSVDirToExcel`, `ReadCSVToString`, `ExcelToCSV`, `ExcelToCSVOptions` and `ExcelDirToCSV` SHALL have a doc comment that starts with its own name.

#### Scenario: The doc comments
- **WHEN** 解析 `csvxl` 的非測試原始檔
- **THEN** 上述七個名稱各自的 doc comment 以自己的名稱開頭

### Requirement: ExcelToCSV and ExcelDirToCSV guard formula-like text by default

`ExcelToCSV` and `ExcelDirToCSV` SHALL prefix cell text that a spreadsheet would run as a formula with an apostrophe, unless `ExcelToCSVOptions.AllowFormulas` is set.

#### Scenario: Formula text with the default options
- **WHEN** 工作表的儲存格為 `=1+1`，以沒有選項的 `ExcelToCSV` 轉出
- **THEN** CSV 中該儲存格為 `'=1+1`

#### Scenario: AllowFormulas
- **WHEN** 同一個儲存格，以 `AllowFormulas` 為 true 的 `ExcelToCSVOptions` 轉出
- **THEN** CSV 中該儲存格為 `=1+1`，沒有單引號前綴

#### Scenario: A plain number is not guarded
- **WHEN** 儲存格為 `-1e309`，以沒有選項的 `ExcelToCSV` 轉出
- **THEN** CSV 中該儲存格仍為 `-1e309`

### Requirement: ExcelToCSV converts only the sheets it is asked for

When `ExcelToCSVOptions.Sheets` is not empty, `ExcelToCSV` SHALL convert only the sheets named there.

#### Scenario: A sheet the workbook does not have
- **WHEN** `Sheets` 含有活頁簿沒有的工作表 `Nope`
- **THEN** 回傳含 `Nope` 的錯誤，且沒有寫出任何 CSV

### Requirement: The old csvxl names keep their meaning for one release

`CsvToExcel`, `AppendCsvToExcel`, `EachCsvToOneExcel` and `ReadCsvToString` SHALL keep their old behaviour with a `Deprecated:` paragraph that names the replacement and says it is removed in 0.4.

#### Scenario: The deprecation notices
- **WHEN** 讀取四個舊名稱（以及 `ExcelToCsv`、`EachExcelToCsv`）的 doc comment
- **THEN** 各自有指名新名稱並寫明 `Removed in 0.4.` 的 `Deprecated:` 段落

#### Scenario: An old name gives the same result
- **WHEN** 以 `CsvToExcel` 與 `CSVToExcel` 轉換同一個 CSV
- **THEN** 兩個工作簿的工作表與儲存格相同

### Requirement: ExcelToCsv and EachExcelToCsv keep their output without the formula guard

`ExcelToCsv` and `EachExcelToCsv` SHALL keep their signatures and output, without the apostrophe guard that `ExcelToCSV` applies.

#### Scenario: The old function does not guard formula text
- **WHEN** 儲存格為 `=1+1`，以 `ExcelToCsv` 轉出
- **THEN** CSV 中該儲存格仍為 `=1+1`，沒有單引號前綴
