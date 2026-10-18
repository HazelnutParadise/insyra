# core-data-input Specification

## Purpose
Defines how raw Go data such as 2D slices becomes a DataTable, so that no value a caller passes in is silently dropped or reshaped on the way in.

## Requirements

### Requirement: A 2D slice becomes a table as wide as its widest row
The system SHALL size the table that `ReadSlice2D` returns by the longest row, pad shorter rows with nil, and keep every value of every row.

#### Scenario: A row longer than the first
- **WHEN** `ReadSlice2D` receives `[][]float64{{1}, {2, 3}}`
- **THEN** the table has two columns
- **AND** the second row holds 2 and 3

#### Scenario: A row shorter than the first
- **WHEN** a row has fewer values than the widest row
- **THEN** its missing cells are nil

#### Scenario: An empty first row
- **WHEN** the first row is empty
- **THEN** the call returns an error
