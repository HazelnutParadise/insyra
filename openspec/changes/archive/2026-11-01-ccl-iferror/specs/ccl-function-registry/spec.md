## ADDED Requirements

### Requirement: IFERROR replaces a failure on a row

`IFERROR(value, valueIfError)` SHALL give `value`, and SHALL compute and give `valueIfError` only when computing `value` fails. An empty result or `NaN` SHALL NOT count as a failure. An undefined function and a column that does not exist SHALL still be errors.

#### Scenario: Dividing by zero

- **WHEN** column B holds 2 and 0 and column A holds 6 and 8
- **THEN** `IFERROR(A / B, -1)` gives 3 and -1

#### Scenario: A mistake in the formula

- **WHEN** an expression is `IFERROR(NOSUCHFN(A), 0)`
- **THEN** the expression fails with an undefined function error
