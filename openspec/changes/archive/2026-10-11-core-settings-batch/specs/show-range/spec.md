## ADDED Requirements

### Requirement: ShowHead and ShowTail display the start and end of data

`ShowHead(n)` and `ShowTail(n)` (and their `To` variants) on `DataTable` and `DataList` SHALL display what `ShowRange(n)` and `ShowRange(-n)` display, and SHALL print an error line for an `n` that is not positive without showing rows.

#### Scenario: The head of a table

- **WHEN** `dt.ShowHead(5)` is called
- **THEN** the output is the same as `dt.ShowRange(5)`

#### Scenario: The tail of a list

- **WHEN** `dl.ShowTail(3)` is called
- **THEN** the output is the same as `dl.ShowRange(-3)`

#### Scenario: Non-positive count

- **WHEN** `dt.ShowHead(0)` or `dt.ShowTail(-2)` is called
- **THEN** an error line is printed and no rows are shown
