## ADDED Requirements

### Requirement: A long Excel name says which name to write

A function name from Excel that CCL has under another name SHALL stay an undefined function error, and the error SHALL name what to write.

#### Scenario: AVERAGE

- **WHEN** an expression is `AVERAGE(A)`
- **THEN** it fails, and the error says to use `AVG`

### Requirement: CEIL and FLOOR round to a multiple

`CEIL(x, m)` SHALL give the smallest multiple of `m` at or above `x`, and `FLOOR(x, m)` the largest at or below, reading both numbers as written. A multiple that is not a positive number SHALL be an error.

#### Scenario: A multiple

- **WHEN** an expression is `CEIL(12, 5)` or `FLOOR(12, 5)`
- **THEN** the result is 15 or 10

#### Scenario: A decimal multiple

- **WHEN** an expression is `CEIL(1.1, 0.1)`
- **THEN** the result is 1.1
