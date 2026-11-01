## ADDED Requirements

### Requirement: CCL builds dates and reads the clock

`DATE(year, month, day)` SHALL give that date, dropping fractions and carrying over numbers past the end of a month or year. `TODAY()` and `NOW()` SHALL give the local clock's date and date and time with no zone, and one call of a table or Parquet method SHALL read one moment on every row, statement and batch.

#### Scenario: A date past the end of the year

- **WHEN** an expression is `DATE(2024, 13, 1)`
- **THEN** the result is 2025-01-01

#### Scenario: An empty argument

- **WHEN** any argument of `DATE` is an empty cell
- **THEN** the result is empty

#### Scenario: One moment per call

- **WHEN** `AddColUsingCCL` evaluates `NOW()` on 5000 rows
- **THEN** every row holds the same value
