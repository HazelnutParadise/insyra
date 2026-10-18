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
