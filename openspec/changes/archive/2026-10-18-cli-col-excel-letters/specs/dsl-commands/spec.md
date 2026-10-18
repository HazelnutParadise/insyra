## ADDED Requirements

### Requirement: col accepts an Excel-style column letter
The system SHALL resolve the selector of `col` as a 0-based position when it is a whole number, otherwise as an exact column name, otherwise as an Excel-style letter that names a column of the table, and SHALL report "column not found" when none applies.

#### Scenario: A letter selects a column
- **WHEN** `col t B` runs on a table with at least two columns and no column named `B`
- **THEN** the second column is saved

#### Scenario: A column name wins over a letter
- **WHEN** a column is named `A` and is not the first column
- **THEN** `col t A` saves the column named `A`

#### Scenario: A letter past the last column
- **WHEN** the letter names a column the table does not have
- **THEN** the command fails with "column not found" and stores nothing
