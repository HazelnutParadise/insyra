## ADDED Requirements

### Requirement: String is the uncoloured view

`DataList.String` and `DataTable.String` SHALL return exactly what their `Show` view writes when laid out for 80 columns without colour, whatever the terminal width and `Config.SetUseColoredOutput` say, so that `fmt`'s `%v` and `%s` print it.

#### Scenario: Colour switched on

- **WHEN** coloured output is on and `dt.String()` is called
- **THEN** the text has no ANSI escape codes and equals `fmt.Sprint(dt)`

### Requirement: A cell holding a table or list is one line

Every view SHALL show a cell that holds a `*DataTable` as `<DataTable>` and one that holds a `*DataList` as `<DataList>`, so a table or list that holds itself can be shown.

#### Scenario: A table holding itself

- **WHEN** a two-row table holds itself in a cell and its String is called
- **THEN** it returns, and the cell reads <DataTable>
