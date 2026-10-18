# sql-unnamed-column-name Specification

## Purpose
A table written with ToSQL names every column, so MySQL and PostgreSQL, which refuse an empty column name, accept a table whose columns have none.

## Requirements

### Requirement: An unnamed column is written under its letter

`ToSQL` and `ToSQLContext` SHALL write a column that has no name under its Excel-style letter, counted among the table's columns without the row-name column, and SHALL refuse the write before anything is created when another column already has that letter as its name, compared without regard to case.

#### Scenario: Unnamed columns
- **WHEN** a table whose first and third columns have no name is written
- **THEN** the table's columns are `A`, the second column's name and `C`

#### Scenario: The letter is taken
- **WHEN** a table's first column has no name and its second is named `a`
- **THEN** the call returns an error naming column `A` and creates no table
