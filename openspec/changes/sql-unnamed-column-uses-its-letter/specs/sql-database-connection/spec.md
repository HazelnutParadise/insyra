## ADDED Requirements

### Requirement: An unnamed column is written under its letter

`ToSQL` SHALL write a column that has no name under its Excel-style letter, counted among the table's columns. When another column already has that letter as its name, compared without case, `ToSQL` SHALL return an error before creating or changing anything.

#### Scenario: Unnamed columns
- **WHEN** a table whose columns are unnamed, `label` and unnamed is written and read back
- **THEN** the columns are named `A`, `label` and `C`

#### Scenario: Letter taken
- **WHEN** the first column is unnamed and the second is named `a`
- **THEN** `ToSQL` returns an error naming column `A`, and no table is created

#### Scenario: Servers accept it
- **WHEN** a table with one unnamed column is written to MySQL or PostgreSQL
- **THEN** the write succeeds, and the column reads back as `A`
