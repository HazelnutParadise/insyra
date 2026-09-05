# Design: add-cli-timeseries-commands

## Context

`cli/commands/timeseries.go` registers `rolling`/`expanding` with a `Forms` list, parses trailing `key value` options in a loop, dispatches reducers by lowercase name, and stores the result through `parseAlias` (`as <var>` or `$result`). `groupby` already maps operator names to `insyra.AggregateOp`. Tests live beside commands (`*_test.go`) and drive `Dispatch` with an `ExecContext` whose `Vars` hold the inputs. `Docs/cli-dsl.md` hand-maintains the Command Groups list and the Full Command Index table; `skills/use-insyra-cli/references/` mirrors the catalogue.

## Goals / Non-Goals

**Goals:** the three primitives reachable from `.isr` and the REPL with the same option names as the Go API.
**Non-Goals:** `EWMCol`/`RollingCol` table forms (use `col` first, as the existing `rolling` does); weights; a resample `as` per column beyond the `:name` suffix.

## Decisions

- **`ewm` is its own command**, not a `rolling` reducer: it has no window and different options. Option names copy `EWMOptions` so docs can point at the Go page.
- **`rolling cov|beta` consume the next positional** as the second series, then continue option parsing from the following index; the existing reducer switch gains two cases that read `ctx.Vars` for the other list.
- **`resample` spec `col:op[:name]`** reuses the operator table `groupby` already parses, so operator names and errors stay identical; the parser splits on `:` and rejects anything but 2 or 3 fields.
- **Errors carry the library message** (`fmt.Errorf("resample: %w", err)`) so the row-numbered non-time error reaches the user.

## Risks / Trade-offs

- [Column names containing `:`] → not supported by the `col:op` syntax; documented, and the error names the offending token.
- [Docs index drift] → the index rows are added by hand as for every prior command; the CLI skill's "run `help <cmd>` first" guidance stands.
