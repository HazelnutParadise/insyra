## ADDED Requirements

### Requirement: A missing or repeated level combination is refused before the grid is allocated

`TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable` and `FriedmanTestFromTable` SHALL find a missing or repeated level combination before they allocate the grid of cells or subjects, and SHALL return the same error as before. A table whose factors are almost all distinct SHALL be refused without allocating a cell for every level combination.

#### Scenario: Two-way table with mostly empty combinations
- **WHEN** `TwoWayANOVAFromTable` is called on a table of 5000 rows whose factors A and B each take 5000 distinct levels, with each row on a diagonal combination
- **THEN** the call returns the error `no observations for A=0, B=1` and allocates less than 64 MiB

#### Scenario: Repeated-measures table with mostly missing combinations
- **WHEN** `RepeatedMeasuresANOVAFromTable` is called on a table of 5000 rows with 5000 distinct subjects and 5000 distinct conditions, each row on a diagonal combination
- **THEN** the call returns an error and allocates less than 64 MiB
