## ADDED Requirements

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
