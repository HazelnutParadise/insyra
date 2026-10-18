# read-write-names Specification

## Purpose
The one-name spellings ReadJSONFile, ToJSONBytes and ToJSONString for reading and writing JSON, with the underscore names kept as Deprecated wrappers until 0.4.

## Requirements

### Requirement: JSON is read and written by one name

`ReadJSONFile`, `DataTable.ToJSONBytes` and `DataTable.ToJSONString` SHALL read a JSON file and write a table as indented JSON. `ReadJSON_File`, `ToJSON_Bytes` and `ToJSON_String` SHALL remain until 0.4, marked Deprecated with a notice that names the new name and says it is removed in 0.4, and SHALL return exactly what the new names return.

#### Scenario: The old reader
- **WHEN** the same file is read with `ReadJSON_File` and with `ReadJSONFile`
- **THEN** the tables are equal

#### Scenario: The old writers
- **WHEN** a table is written with `ToJSON_Bytes` and with `ToJSONBytes`, and with `ToJSON_String` and with `ToJSONString`, for both values of `useColNames`
- **THEN** each pair is identical

#### Scenario: The notices
- **WHEN** the doc comments of the three old names are read
- **THEN** each has a `Deprecated:` paragraph naming its replacement and saying it is removed in 0.4
