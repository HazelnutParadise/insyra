# ccl-function-registry Specification

## Purpose
Defines how a program outside the module adds its own CCL functions through `engine/ccl`: the three kinds of function it can register, and how the one process-wide registry behaves.

## Requirements
### Requirement: engine/ccl registers every kind of CCL function

`engine/ccl` SHALL let a caller register a scalar function (`RegisterFunction`), an aggregate function (`RegisterAggregateFunction`) and a sequence function (`RegisterSequenceFunction`, with the type `SeqFunc`), each into the registry the evaluator reads.

#### Scenario: A sequence function from outside the module
- **WHEN** a caller registers `ENGINETESTREVERSE` with `RegisterSequenceFunction` and evaluates `ENGINETESTREVERSE(A)` over a column `[1, 2, 3]`
- **THEN** the result is the column `[3, 2, 1]`

### Requirement: The registry's behaviour is documented and holds

The documentation of `engine/ccl` SHALL state that there is one function registry for the process, that registration and evaluation are safe from any number of goroutines, that names are matched in any letter case, and that registering a name again replaces the function of the same kind, a built-in's included.

#### Scenario: A name registered twice replaces the first
- **WHEN** a caller registers a scalar function under a name, then registers another under the same name, and evaluates an expression that calls it
- **THEN** the second function is the one that runs, even when the name is a built-in's

### Requirement: No-op functions say they do nothing

`ResetEvalDepth` and `ResetFuncCallDepth` SHALL carry a `Deprecated:` notice saying they do nothing, and SHALL stay callable until they are removed.

#### Scenario: Reading the documentation
- **WHEN** a caller reads the doc comment of `ResetEvalDepth` or `ResetFuncCallDepth`
- **THEN** it contains `Deprecated: it does nothing`

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
### Requirement: IFERROR replaces a failure on a row

`IFERROR(value, valueIfError)` SHALL give `value`, and SHALL compute and give `valueIfError` only when computing `value` fails. An empty result or `NaN` SHALL NOT count as a failure. An undefined function and a column that does not exist SHALL still be errors.

#### Scenario: Dividing by zero

- **WHEN** column B holds 2 and 0 and column A holds 6 and 8
- **THEN** `IFERROR(A / B, -1)` gives 3 and -1

#### Scenario: A mistake in the formula

- **WHEN** an expression is `IFERROR(NOSUCHFN(A), 0)`
- **THEN** the expression fails with an undefined function error
