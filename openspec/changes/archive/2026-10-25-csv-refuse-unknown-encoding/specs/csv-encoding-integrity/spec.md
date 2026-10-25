## ADDED Requirements

### Requirement: An explicit encoding nothing decodes is an error in every CSV reader

`ReadCSV`, `ReadCSV_FileWithOptions`, `StreamCSV`, `csvxl.CsvToExcel` and `csvxl.AppendCsvToExcel` SHALL return an error listing the supported encodings when given an encoding name that is neither in the decoder table nor matched by its substring rules, and SHALL NOT read the bytes as they are. A charset the auto-detector reports SHALL be read as before.

#### Scenario: A mistyped name

- **WHEN** a Big5 file is read with the encoding "big6"
- **THEN** the call returns an error that names "big6" and the supported encodings

#### Scenario: A converter writes nothing

- **WHEN** CsvToExcel is given the encoding "big6"
- **THEN** it returns the error and creates no output file

### Requirement: Windows code page names for Big5 and GBK are accepted

The decoder table SHALL accept `cp950` as Big5 and `cp936` as GB18030, as it accepts `cp932` and `cp949`.

#### Scenario: cp950

- **WHEN** a Big5 file is read with the encoding "cp950"
- **THEN** its text is decoded from Big5
