## ADDED Requirements

### Requirement: Quantile is the type-7 quantile on the 0..1 scale

`DataList.Quantile(p)` SHALL return the R type-7 quantile of the list's numeric values for `p` in `[0, 1]`, skipping `nil` and `NaN` cells. For a list without NaN or non-numeric cells it SHALL equal `Percentile(p*100)`, and `Quartile(q)` SHALL equal `Quantile(q/4)`.

#### Scenario: Matches pandas
- **WHEN** `Quantile` is called with 0, 0.25, 0.5, 0.75 and 1 on `[1, 2, 3, 4, 10]`
- **THEN** it returns 1, 2, 3, 4 and 10, which is what pandas' `Series.quantile` returns for the same values

#### Scenario: Missing cells
- **WHEN** `Quantile(0.25)` is called on `[1, 2, 3, NaN]`
- **THEN** it returns 1.5

### Requirement: Quantile refuses what it cannot answer

`Quantile` SHALL record a warning and return `NaN`, without panicking, for a `p` that is `NaN` or outside `[0, 1]`, for a cell that is neither a number nor missing, and for a list with no numeric values.

#### Scenario: p out of range
- **WHEN** `Quantile(1.1)` is called
- **THEN** it returns `NaN` and `Err()` is not nil

#### Scenario: A text cell
- **WHEN** `Quantile(0.5)` is called on `[1, "x", 3]`
- **THEN** it returns `NaN` and `Err()` is not nil

### Requirement: The CLI prints a quantile

The command `quantile <var> <p>` SHALL print `Quantile(p)` of the DataList in `<var>`, for `p` from 0 to 1.

#### Scenario: Same number as percentile
- **WHEN** `quantile x 0.75` and `percentile x 75` run on `[1, 2, 3, 4, 10]`
- **THEN** both print 4
