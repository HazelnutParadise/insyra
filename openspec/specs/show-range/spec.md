# show-range Specification

## Purpose
How the display functions read the range they are given: a count, or a start and an end. A range they cannot read is reported instead of silently showing everything, and ShowHead and ShowTail give the common ranges plain names.
## Requirements
### Requirement: A range the display cannot read is reported

`ShowRange` and `ShowTypesRange`, on DataList and DataTable, SHALL print an error line instead of the data when given more than two values, a first value that is not an int, or an end that is neither an int nor nil. `ShowHead(n)` and `ShowTail(n)` SHALL display what `ShowRange(n)` and `ShowRange(-n)` display, and SHALL print an error line for an `n` that is not positive.

#### Scenario: A third value

- **WHEN** `dt.ShowRange(1, 2, 3)` is called
- **THEN** an error line is printed and no rows are shown

#### Scenario: The head of a table

- **WHEN** `dt.ShowHead(5)` is called
- **THEN** the output is the same as `dt.ShowRange(5)`

### Requirement: A table view is sized by the rows it prints

`ShowRange`, `Show` and `ShowTypesRange` on a DataTable SHALL compute column widths and row labels from the rows they print and SHALL NOT format cells outside them. A cell or row name outside the printed rows SHALL NOT change the output. When every row is printed, the output SHALL be what it was before this requirement.

#### Scenario: A wide cell below the range

- **WHEN** two tables differ only in row 7, one holding a 60-character text and a long row name, and each is shown with `ShowRange(5)`
- **THEN** the two outputs are identical

#### Scenario: The default view of a long table

- **WHEN** two 70-row tables differ only in row 40, which the default view does not print, and each is shown with `Show`
- **THEN** the two outputs are identical

### Requirement: Show takes an exported Showable

`Show` SHALL take its object as the exported interface `Showable`, whose one method is `ShowRange(startEnd ...any)`, so a caller can name the type in its own signatures.

#### Scenario: A slice of things to show

- **WHEN** a caller builds a `[]insyra.Showable` holding a DataTable and a DataList and passes each to `Show`
- **THEN** it compiles and shows both

### Requirement: A view with no range prints up to 60 rows whole

`Show`, and `ShowRange` and `ShowTypesRange` called with no range, on DataTable and DataList, SHALL print every row of a view of up to 60 rows, and SHALL print only the first 20 and the last 5 rows of a longer one, with a line between them saying that rows were left out. A view given an explicit range SHALL print every row of the range.

#### Scenario: Sixty rows

- **WHEN** a 60-row table or list is shown with no range
- **THEN** all 60 rows are printed

#### Scenario: Sixty-one rows

- **WHEN** a 61-row table or list is shown with no range
- **THEN** rows 0 to 19 and 56 to 60 are printed, and row 20 is not

