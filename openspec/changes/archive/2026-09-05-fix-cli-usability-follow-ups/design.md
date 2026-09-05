# Design: fix-cli-usability-follow-ups

## Context

`datatable_from_sql.go` holds `dateParseLayouts` and a per-cell loop that tries them when `ReadSQLOptions.ParseDates` names a column; nothing exposes that to a loaded CSV. `cli/env/state.go` decodes with `UseNumber` and applies `coerceEnvNumber` only while rebuilding DataList elements, so a top-level scalar stays `json.Number`. `CommandHandler.DisableFlagParsing` exists and `show` already uses it; `BuildCobraCommands` passes it through; `help` is a registry command, not a Cobra flag.

## Decisions

- **One shared `ParseDates` on `DataList`, table wrapper on top.** The SQL path delegates to it so there is one layout list and one nil rule. Unparsable cells become `nil` rather than staying strings: a half-converted column would defeat `Resample`'s type check in a confusing way, and `nil` is what the window commands already emit for the unusable.
- **Coerce scalars at load, not at use.** Every reader of `ctx.Vars` benefits; the change is one loop in `LoadState`.
- **`DisableFlagParsing` on the three `<values...>` commands only.** Commands whose arguments are variable names never see a leading `-`; the docs note `--` for anything else.

## Risks / Trade-offs

- [A string column that is dates in a layout not in the default list] → `layout` option; the error path is `nil` cells, which `Resample` then reports by row.
- [`DisableFlagParsing` hides `--help`] → `help newdl` is the documented route and is tested.
