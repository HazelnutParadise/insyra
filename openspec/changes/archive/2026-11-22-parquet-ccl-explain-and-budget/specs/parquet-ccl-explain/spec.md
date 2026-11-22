## ADDED Requirements

### Requirement: Explain plans a filter without evaluating rows

`ExplainFilterWithCCL` SHALL return the `CCLPlan` `FilterWithCCL` would follow for the same file and expression, reading only the file's metadata and evaluating no row. Its `Mode` SHALL be `CCLWholeColumns` exactly when the call loads whole columns, with the reason the call has for it, and `HeldColumns` SHALL name those columns.

#### Scenario: An aggregate the batches can compute
- **WHEN** a 2,500-row file is explained for `A > AVG(A)`
- **THEN** the mode is `CCLStreamed`, `MinDataPasses` is 2, and `FilterWithCCL` reads the file's data twice

#### Scenario: MEDIAN needs whole columns
- **WHEN** a file is explained for `A > MEDIAN(A)`
- **THEN** the mode is `CCLWholeColumns`, a reason names MEDIAN, `HeldColumns` is the column A, and the pass count equals the passes `FilterWithCCL` makes

### Requirement: Explain separates exact counts from estimates

`MinDataPasses` and `HeldCells` SHALL be exact for a run that succeeds. `MayReadMore` SHALL be true, with a reason, when only the data decides whether the call reads more or loads whole columns. `EstimatedHeldBytes` SHALL be documented as an estimate from the file's metadata.

#### Scenario: A sequence period computed from the data
- **WHEN** a file is explained for `LAG(A, COUNT(A))`
- **THEN** `MayReadMore` is true and a reason names the period

### Requirement: Explain refuses what the call refuses

An expression that does not compile, or names a column the file does not have where the call checks it, SHALL be an error from Explain as it is from the call. A plan SHALL NOT promise that the call succeeds on the data.

#### Scenario: A syntax error
- **WHEN** `ExplainFilterWithCCL` is given `A >`
- **THEN** it returns the error `FilterWithCCL` returns

### Requirement: ApplyCCL is explained statement by statement

`ExplainApplyCCL` SHALL return one plan per statement, in order, each counting the passes that statement adds before the file is written. A statement that writes nothing SHALL be reported as skipped.

#### Scenario: A script with a whole-column statement
- **WHEN** a script `NEW('B') = A * 2; NEW('C') = A - MEDIAN(A)` is explained
- **THEN** the first plan is `CCLStreamed` and the second is `CCLWholeColumns` holding A

### Requirement: FilterWithCCL can cap the rows it keeps

With `FilterWithCCLOptions.MaxRows` above zero, `FilterWithCCL` SHALL stop reading at the first kept row past the limit and return `nil` and an error matching `ErrCCLRowLimit` that names the limit. A negative `MaxRows`, or more than one options value, SHALL be an error before anything is read.

#### Scenario: A filter that keeps every row
- **WHEN** a 2,500-row file is filtered with `A > -1` and `MaxRows` 1,000
- **THEN** the result is `nil` and the error matches `ErrCCLRowLimit`

#### Scenario: Within the limit
- **WHEN** the same filter keeps 999 rows under `MaxRows` 1,000
- **THEN** the call returns those rows and no error

### Requirement: FilterWithCCL can refuse whole columns

With `RefuseWholeColumns`, an expression that would load whole columns SHALL fail before any data pass with an error matching `ErrCCLWholeColumnsRefused` that carries the reasons.

#### Scenario: MEDIAN refused
- **WHEN** a file is filtered with `A > MEDIAN(A)` and `RefuseWholeColumns`
- **THEN** the error matches `ErrCCLWholeColumnsRefused` and no data pass was made
