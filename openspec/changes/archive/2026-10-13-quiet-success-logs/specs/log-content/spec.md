## ADDED Requirements

### Requirement: A success the return value reports is not logged

A function whose nil error already reports that it succeeded SHALL NOT also log that success. This applies to `csvxl.CSVToExcel`, `AppendCSVToExcel`, `ExcelToCSV` and `ExcelDirToCSV`, to `plot.SaveHTML` and `SavePNG`, to `gplot.SaveChart`, and to `py.PipInstall`, `PipUninstall` and `PipList`.

#### Scenario: Saving a chart
- **WHEN** a logger is set at Info level and `plot.SaveHTML` or `gplot.SaveChart` writes a chart and returns nil
- **THEN** the call writes no log record

#### Scenario: Converting files
- **WHEN** a logger is set at Info level and `csvxl.CSVToExcel`, `AppendCSVToExcel`, `ExcelToCSV` or `ExcelDirToCSV` converts a UTF-8 file and returns nil
- **THEN** the only records are the encoding-detection lines
- **AND** no record comes from the converting function

#### Scenario: Functions no test can run here
- **WHEN** the source of `plot.SavePNG`, `py.PipInstall`, `py.PipUninstall` and `py.PipList` is parsed
- **THEN** none of them calls `insyra.LogInfo`

### Requirement: Info lines say what the return value does not

insyra SHALL keep logging at Info a guess it made about the data, a change made outside the call's own result, the progress of a long operation, and a switch changing state.

#### Scenario: A guessed encoding is still logged
- **WHEN** a logger is set at Info level and `ReadCSVFile` reads a UTF-8 file without a given encoding
- **THEN** one Info record names the encoding it detected
