# show-range Specification

## Purpose
What the table and list views print: `ShowHead` and `ShowTail`, how a view sizes its columns from the rows it prints, and the exported `Showable` interface `Show` takes.
## Requirements
### Requirement: ShowHead and ShowTail display the start and end of data

`ShowHead(n)` and `ShowTail(n)` (and their `To` variants) on `DataTable` and `DataList` SHALL display what `ShowRange(n)` and `ShowRange(-n)` display, and SHALL print an error line for an `n` that is not positive without showing rows.

#### Scenario: The head of a table

- **WHEN** `dt.ShowHead(5)` is called
- **THEN** the output is the same as `dt.ShowRange(5)`

#### Scenario: The tail of a list

- **WHEN** `dl.ShowTail(3)` is called
- **THEN** the output is the same as `dl.ShowRange(-3)`

#### Scenario: Non-positive count

- **WHEN** `dt.ShowHead(0)` or `dt.ShowTail(-2)` is called
- **THEN** an error line is printed and no rows are shown

### Requirement: A table view is sized by the rows it prints

`ShowRange`, `Show` and `ShowTypesRange` on a DataTable SHALL compute column widths and row labels from the rows they print and SHALL NOT format cells outside them. A cell or row name outside the printed rows SHALL NOT change the output. When a table has more than 25 rows and is shown without an explicit range, it SHALL print the first 20 and last 5 rows. When every row is printed, the output SHALL be what it was before this requirement.

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

### Requirement: String is the uncoloured view

`DataList.String` and `DataTable.String` SHALL return exactly what their `Show` view writes when laid out for 80 columns without colour, whatever the terminal width and `Config.SetUseColoredOutput` say, so that `fmt`'s `%v` and `%s` print it.

#### Scenario: Colour switched on

- **WHEN** coloured output is on and `dt.String()` is called
- **THEN** the text has no ANSI escape codes and equals `fmt.Sprint(dt)`

### Requirement: A cell holding a table or list is one line

Every view SHALL show a cell that holds a `*DataTable` as `DataTable(rows x cols)` and one that holds a `*DataList` as `DataList(n)`, so a table or list that holds itself can be shown.

#### Scenario: A table holding itself

- **WHEN** a two-row table holds itself in a cell and its String is called
- **THEN** it returns, and the cell reads DataTable(2x1)

