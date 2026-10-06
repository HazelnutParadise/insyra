# engine-ccl-api Specification

## Purpose
What `engine/ccl` promises a program that compiles CCL and evaluates it against its own data: a `Context` whose methods do not change, compiled nodes that cannot be confused with other values, and a `MapContext` that can only be changed through its checked methods.
## Requirements
### Requirement: Context has a fixed method set

`engine/ccl.Context` SHALL keep exactly the methods `GetAllData`, `GetCell`, `GetCellByName`, `GetCol`, `GetColByName`, `GetColCount`, `GetColData`, `GetColDataByName`, `GetColIndexByName`, `GetCurrentRow`, `GetRowAt`, `GetRowCount`, `GetRowIndex`, `GetRowIndexByName` and `SetRowIndex`. A capability CCL gains later SHALL come as a separate interface the evaluator checks for.

#### Scenario: The method set
- **WHEN** the methods of `engine/ccl.Context` are listed
- **THEN** they are exactly those fifteen

### Requirement: CCLNode holds only a compiled node

`engine/ccl.CCLNode` SHALL be an opaque type with no exported fields, made only by the package's compile and node functions. `Evaluate`, `EvaluateStatement` and `Bind` SHALL return an error for the zero `CCLNode`.

#### Scenario: The zero node
- **WHEN** `Evaluate(ccl.CCLNode{}, ctx)` is called
- **THEN** it returns an error

#### Scenario: A compiled node
- **WHEN** `CompileExpression("A + B")` succeeds and its node is evaluated on row 2 of a MapContext with `A` = 1, 2, 3 and `B` = 10, 20, 30
- **THEN** the node is not the zero `CCLNode` and the result is 33

### Requirement: MapContext changes only through its checked methods

`engine/ccl.MapContext` SHALL have no exported fields. It SHALL be made by `NewMapContext` and SHALL implement `Context`, refusing a row that is not there in `SetRowIndex`.

#### Scenario: An out-of-range row
- **WHEN** `SetRowIndex(99)` is called on a MapContext of three rows
- **THEN** it returns an error

