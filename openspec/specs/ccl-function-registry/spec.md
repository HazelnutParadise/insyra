# ccl-function-registry Specification

## Purpose
What `engine/ccl` lets code outside the module register into CCL's process-wide function registry, and how that registry behaves when several goroutines register and evaluate at once.
## Requirements
### Requirement: engine/ccl registers every kind of CCL function

`engine/ccl` SHALL let a caller register a scalar function (`RegisterFunction`), an aggregate function (`RegisterAggregateFunction`) and a sequence function (`RegisterSequenceFunction`, with the type `SeqFunc`), each into the registry the evaluator reads.

#### Scenario: A sequence function from outside the module
- **WHEN** a caller registers `ENGINETESTREVERSE` with `RegisterSequenceFunction` and evaluates `ENGINETESTREVERSE(A)` over a column `[1, 2, 3]`
- **THEN** the result is the column `[3, 2, 1]`

### Requirement: The registry's behaviour is documented and holds

The documentation of `engine/ccl` SHALL state that there is one function registry for the process, that registration and evaluation are safe from any number of goroutines, and that names are matched in any letter case. Registering an aggregate or sequence function SHALL change the function and the mark that makes the name the caller's in one step, so that no reader sees one without the other.

#### Scenario: Registering while reading

- **WHEN** one goroutine repeatedly registers a built-in and then a caller's function under the same aggregate or sequence name while another reads the registry
- **THEN** the reader never sees the caller's function without the caller's mark

### Requirement: The documentation states the replacement rule per function kind

The documentation of `engine/ccl` SHALL state that registering a name again replaces the function of the same kind, a built-in of that kind included, and MUST NOT suggest that it replaces a function of another kind.

#### Scenario: Reading the scalar registration

- **WHEN** a caller reads the doc comment of `RegisterFunction`
- **THEN** it says that a later scalar registration replaces this one

### Requirement: The documentation states the evaluation order

The documentation of `engine/ccl` SHALL state that evaluation checks sequence functions first, then aggregate functions, then scalar functions, and the evaluator MUST check them in that order.

#### Scenario: Reading the aggregate registration

- **WHEN** a caller reads the doc comment of `RegisterAggregateFunction`
- **THEN** it says that evaluation checks sequence, aggregate, then scalar functions

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

