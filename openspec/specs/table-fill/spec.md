# table-fill Specification

## Purpose
What a table-level fill does with a column it cannot fill: a column the caller named is reported, so a fill never looks done when it was not, while filling every column still skips what it cannot handle. Table interpolation extrapolates on request, as list interpolation does.

## Requirements
### Requirement: A named column a table fill cannot fill is an error

A table-level `FillWithMean`, `FillWithMedian`, `FillByInterpolation` or `FillWithMode` SHALL record an error naming the column and why it cannot be filled when the caller named that column, and SHALL still fill the other named columns. When no column is named, a column the fill cannot handle SHALL be skipped without an error.

#### Scenario: A text column named for the mean

- **WHEN** `dt.FillWithMean(Name("price"), Name("city"))` runs and `city` holds text
- **THEN** `price` is filled and `dt.Err()` names `city` and its string values

#### Scenario: No column named

- **WHEN** `dt.FillWithMean()` runs on a table with a number column and a text column
- **THEN** the number column is filled, the text column is unchanged, and no error is recorded

### Requirement: Table interpolation can extrapolate

`DataTable.FillByInterpolation(extrapolate, cols...)` SHALL fill before the first and after the last observed value when `extrapolate` is true, as `DataList.FillByInterpolation(true)` does.

#### Scenario: Gaps at both ends

- **WHEN** a column `[nil, 2, 3, nil]` is interpolated with `extrapolate` true
- **THEN** it becomes `[1, 2, 3, 4]`

