## MODIFIED Requirements

### Requirement: The old names keep their old meaning for one release

`ReadCSV_File`, `ReadCSV_FileWithOptions`, `ReadCSV_String`, `ReadCSV_StringWithOptions` and `ToCSVWithOptions` SHALL remain, marked Deprecated, and SHALL return what they returned before, positional bools included.

#### Scenario: The old reader without a header

- **WHEN** `ReadCSV_File(path, false, false)` reads a two-line file
- **THEN** both lines are data rows
