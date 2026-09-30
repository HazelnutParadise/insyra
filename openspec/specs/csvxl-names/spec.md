# csvxl-names Specification

## Purpose
The names of the `csvxl` functions: they follow Go's initialism convention, every exported identifier's doc comment starts with its name, the old `Csv` spellings stay one release as Deprecated wrappers, and an encoding no decoder handles is refused before any file is read.

## Requirements

### Requirement: csvxl names follow Go's initialism convention

`csvxl` SHALL export `CSVToExcel`, `AppendCSVToExcel`, `ExcelToCSV`, `ExcelToCSVOptions`, `CSVDirToExcel`, `ExcelDirToCSV` and `ReadCSVToString`, each taking the parameters and doing what its old spelling did. Every exported identifier of the package SHALL have a doc comment that starts with its name, and the package comment SHALL start with `Package csvxl`.

#### Scenario: The doc comments
- **WHEN** 解析 `csvxl` 的非測試原始檔
- **THEN** 每個匯出的函式、型別與常數都有以自己名稱開頭的 doc comment，套件註解以 `Package csvxl` 開頭

#### Scenario: A directory of CSV files
- **WHEN** 目錄裡有兩個 CSV，呼叫 `CSVDirToExcel(dir, out)`
- **THEN** `out` 是一個工作簿，每個 CSV 各一張工作表

### Requirement: The old csvxl names keep their meaning for one release

`CsvToExcel`, `AppendCsvToExcel`, `ExcelToCsv`, `EachCsvToOneExcel`, `EachExcelToCsv` and `ReadCsvToString` SHALL remain as wrappers of the new functions, and `ExcelToCsvOptions` SHALL remain as an alias of `ExcelToCSVOptions`. Each SHALL have a doc comment whose `Deprecated:` paragraph names its replacement and says it is removed in the release after the one that deprecated it.

#### Scenario: The deprecation notices
- **WHEN** 讀取七個舊名稱的 doc comment
- **THEN** 各自有指名新名稱的 `Deprecated:` 段落

#### Scenario: An old name gives the same result
- **WHEN** 以 `CsvToExcel` 與 `CSVToExcel` 轉換同一個 CSV
- **THEN** 兩個工作簿的工作表與儲存格相同

### Requirement: The conversions refuse an encoding they cannot decode

`CSVToExcel`, `AppendCSVToExcel`, `CSVDirToExcel` and `ReadCSVToString` SHALL return an error naming an encoding that no decoder handles, and SHALL NOT write a workbook for it: `CSVToExcel` and `CSVDirToExcel` create no file, and `AppendCSVToExcel` leaves the workbook as it was. The name SHALL be checked before any file is read, so it is refused when there is no file to read and named once when there are several.

#### Scenario: A name no decoder owns
- **WHEN** 以 `"klingon-1"` 呼叫 `CSVToExcel`
- **THEN** 回傳含 `klingon-1` 的錯誤，輸出路徑沒有檔案

#### Scenario: No files to read
- **WHEN** 以 `"klingon-1"` 呼叫 `CSVToExcel(nil, nil, out, ...)`，或以空目錄呼叫 `CSVDirToExcel`
- **THEN** 回傳含 `klingon-1` 的錯誤，輸出路徑沒有檔案

#### Scenario: Several files
- **WHEN** 以 `"klingon-1"` 轉換三個 CSV
- **THEN** 錯誤訊息裡 `klingon-1` 只出現一次

#### Scenario: Appending with an unknown encoding
- **WHEN** 以 `"klingon-1"` 呼叫 `AppendCSVToExcel` 到既有工作簿
- **THEN** 回傳含 `klingon-1` 的錯誤，工作簿的位元組不變
