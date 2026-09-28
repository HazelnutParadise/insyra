## ADDED Requirements

### Requirement: A view with no range prints up to 60 rows whole

`Show`, and `ShowRange` and `ShowTypesRange` called with no range, on DataTable and DataList, SHALL print every row of a view of up to 60 rows, and SHALL print only the first 20 and the last 5 rows of a longer one, with a line between them saying that rows were left out. A view given an explicit range SHALL print every row of the range.

#### Scenario: Sixty rows

- **WHEN** a 60-row table or list is shown with no range
- **THEN** all 60 rows are printed

#### Scenario: Sixty-one rows

- **WHEN** a 61-row table or list is shown with no range
- **THEN** rows 0 to 19 and 56 to 60 are printed, and row 20 is not

## MODIFIED Requirements

### Requirement: A table view is sized by the rows it prints

`ShowRange`, `Show` and `ShowTypesRange` on a DataTable SHALL compute column widths and row labels from the rows they print and SHALL NOT format cells outside them. A cell or row name outside the printed rows SHALL NOT change the output. When every row is printed, the output SHALL be what it was before this requirement.

#### Scenario: A wide cell below the range

- **WHEN** two tables differ only in row 7, one holding a 60-character text and a long row name, and each is shown with `ShowRange(5)`
- **THEN** the two outputs are identical

#### Scenario: The default view of a long table

- **WHEN** two 70-row tables differ only in row 40, which the default view does not print, and each is shown with `Show`
- **THEN** the two outputs are identical
