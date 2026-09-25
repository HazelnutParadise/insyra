## ADDED Requirements

### Requirement: An encoding argument means the same in csvxl as in the core readers

`csvxl`'s CSV readers SHALL detect the encoding when given an empty string or `"auto"` in any case, as the core CSV readers do.

#### Scenario: Uppercase AUTO on a Big5 file

- **WHEN** `ReadCsvToString(path, "AUTO")` reads a Big5 file
- **THEN** the text is decoded from Big5 instead of the call failing
