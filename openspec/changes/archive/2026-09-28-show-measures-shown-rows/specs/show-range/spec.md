## ADDED Requirements

### Requirement: A table view is sized by the rows it prints

`ShowRange`, `Show` and `ShowTypesRange` on a DataTable SHALL compute column widths and row labels from the rows they print and SHALL NOT format cells outside them. A cell or row name outside the printed rows SHALL NOT change the output. When every row is printed, the output SHALL be what it was before this requirement.

#### Scenario: A wide cell below the range

- **WHEN** two tables differ only in row 7, one holding a 60-character text and a long row name, and each is shown with `ShowRange(5)`
- **THEN** the two outputs are identical

#### Scenario: The default view of a long table

- **WHEN** two 30-row tables differ only in row 22, which the default view does not print, and each is shown with `Show`
- **THEN** the two outputs are identical

### Requirement: Show takes an exported Showable

`Show` SHALL take its object as the exported interface `Showable`, whose one method is `ShowRange(startEnd ...any)`, so a caller can name the type in its own signatures.

#### Scenario: A slice of things to show

- **WHEN** a caller builds a `[]insyra.Showable` holding a DataTable and a DataList and passes each to `Show`
- **THEN** it compiles and shows both
