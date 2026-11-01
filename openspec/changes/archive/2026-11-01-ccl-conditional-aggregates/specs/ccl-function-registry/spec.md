## ADDED Requirements

### Requirement: Conditional aggregates take the condition as an expression

`COUNTIF(condition)`, `SUMIF(condition, values)` and `AVGIF(condition, values)` SHALL read the condition as an expression evaluated per row, and SHALL NOT parse a condition out of text. A row whose condition is empty SHALL NOT hold, a condition that is neither true nor false on a row SHALL be an error naming the row, and a condition that does not depend on the row SHALL hold for every row or none.

#### Scenario: A condition on a column

- **WHEN** column A holds 10, -3, 25, nil and 7, and column B holds 1, 2, 3, 4 and nil
- **THEN** `COUNTIF(A > 0)` is 3, `SUMIF(A > 0, B)` is 4 and `AVGIF(A > 0, B)` is 2

#### Scenario: A condition that does not depend on the row

- **WHEN** a table has 5 rows
- **THEN** `COUNTIF(TRUE)` is 5 and `COUNTIF(FALSE)` is 0

#### Scenario: A condition that is not true or false

- **WHEN** column C holds the word `maybe`
- **THEN** `COUNTIF(C)` is an error that names the value and its row
