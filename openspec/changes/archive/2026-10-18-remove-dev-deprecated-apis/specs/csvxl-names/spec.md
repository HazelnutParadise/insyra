## REMOVED Requirements

### Requirement: The old csvxl names keep their meaning for one release
**Reason**: The Deprecated `Csv` spellings (`CsvToExcel`, `AppendCsvToExcel`, `ExcelToCsv`, `EachCsvToOneExcel`, `EachExcelToCsv`, `ReadCsvToString`) and the `ExcelToCsvOptions` alias were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use `CSVToExcel`, `AppendCSVToExcel`, `ExcelToCSV`, `CSVDirToExcel`, `ExcelDirToCSV`, `ReadCSVToString` and `ExcelToCSVOptions`.
