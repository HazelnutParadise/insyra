## ADDED Requirements

### Requirement: The largest parsed column index does not overflow

`FilterColsByColIndexLessThanOrEqualTo` SHALL accept the largest column letter that `ParseColIndex` reads and MUST NOT panic on it, and SHALL keep every column of the table when the index is past the table's last column.

#### Scenario: The largest parsed letter

- **WHEN** `FilterColsByColIndexLessThanOrEqualTo` is called with the letter for the largest `int` column number on a four-column table
- **THEN** the result holds all four columns and no panic occurs
