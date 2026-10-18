## MODIFIED Requirements

### Requirement: csvxl exports the CSV-spelled names

`csvxl` SHALL export `CSVToExcel`, `AppendCSVToExcel`, `CSVDirToExcel` and `ReadCSVToString`. `CSVDirToExcel` and `ReadCSVToString` SHALL take the arguments and do what their old spellings do. `CSVToExcel` and `AppendCSVToExcel` SHALL take a `[]SheetFromCSV` in place of the old spellings' two slices, as `csvxl-sheet-pairing` describes, and otherwise do what their old spellings do.

#### Scenario: A directory of CSV files
- **WHEN** 目錄裡有兩個 CSV，呼叫 `CSVDirToExcel(dir, out)`
- **THEN** `out` 是一個工作簿，每個 CSV 各一張工作表

### Requirement: The old csvxl names keep their meaning for one release

`CsvToExcel`, `AppendCsvToExcel`, `EachCsvToOneExcel` and `ReadCsvToString` SHALL keep their old behaviour with a `Deprecated:` paragraph that names the replacement and says it is removed in 0.4.

#### Scenario: The deprecation notices
- **WHEN** 讀取四個舊名稱（以及 `ExcelToCsv`、`EachExcelToCsv`）的 doc comment
- **THEN** 各自有指名新名稱並寫明 `Removed in 0.4.` 的 `Deprecated:` 段落

#### Scenario: An old name gives the same result
- **WHEN** 以 `CsvToExcel(files, names, out)` 與把同樣檔名和工作表名稱寫成 `SheetFromCSV` 的 `CSVToExcel` 轉換同一個 CSV
- **THEN** 兩個工作簿的工作表與儲存格相同
