# Design: stats-result-strings

## Decisions

### One renderer over the fields, not thirty hand-written layouts

The review asked for a consistent `String()`. Thirty-three hand-written methods would drift apart the first time a field is added to one of them. The renderer walks a result's exported fields with reflection, in declaration order, so every result prints with the same rules, a new field appears without anyone remembering to add it, and the printed names are the names a program reads. Each type's `String` is one line naming its title.

The rules are the ones the rest of the library already uses, so nothing new has to be learned:

- numbers go through `internal/utils.FloatText`, the rule every text output of insyra follows since v0.4;
- a list or a table prints whole up to 60 entries and as the first 20 and last 5 past that, the table views' rule;
- a `nil` pointer means "does not apply" (the rule `stats-test-results` set), so its line is left out rather than printed as `<nil>`.

A field of a type the renderer has no rule for falls back to `fmt.Sprint`, so it still prints.

### One line per field, a grid per table

Every field is one line, `  Name: value`, including slices, arrays and nested structs such as an ANOVA component (`{SumOfSquares: 12.5, DF: 2, …}`), which keeps the output greppable and the field names visible. A table is the one exception: a factor loading matrix or a contingency table on one line cannot be read, so a table field prints as a grid under its name, indented, with its row names and column names. The grid is built here from the table's names and cells instead of through `DataTable.ShowTo`, because the view adds colour codes when standard output is a terminal and sizes columns to the terminal's width, and `String` has to be the same text everywhere.

### Why not remove `Show`

`Show` does the same thing as `fmt.Println(res)`, so removing the two that exist was an option. It was rejected because `Show` is how insyra's own types print, and users call it on results the way they call it on tables. It is kept, added to every result type, and defined through `String`, so it is a convenience and not a second layout.

### `FactorAnalysisResult.Show` keeps its range

`Show(startEndRange ...any)` passes a row range to each of the result's tables, which matters for the factor scores, one row per observation. Dropping the parameter would remove a capability. With no range it prints `String()` like every other result; with a range it shows each table through the range, as it did.

## Risks / Trade-offs

- Code that parsed the output of the two existing `Show` methods sees a different layout. The changelog says so.
- Reflection costs more than hand-written formatting. `String` is for people and logs, not for a hot path.
