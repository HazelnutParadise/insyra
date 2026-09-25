## ADDED Requirements

### Requirement: A range the display cannot read is reported

`ShowRange` and `ShowTypesRange`, on DataList and DataTable, SHALL print an error line instead of the data when given more than two values, a first value that is not an int, or an end that is neither an int nor nil. `ShowHead(n)` and `ShowTail(n)` SHALL display what `ShowRange(n)` and `ShowRange(-n)` display, and SHALL print an error line for an `n` that is not positive.

#### Scenario: A third value

- **WHEN** `dt.ShowRange(1, 2, 3)` is called
- **THEN** an error line is printed and no rows are shown

#### Scenario: The head of a table

- **WHEN** `dt.ShowHead(5)` is called
- **THEN** the output is the same as `dt.ShowRange(5)`
