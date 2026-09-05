# Proposal: fix-cli-usability-follow-ups

## Why

Three gaps recorded in `AGENTS.md` Follow-ups on 2026-09-05 while exposing the time-series and quant work in the CLI. Each blocks a natural CLI path: a CSV-loaded date column is a string, so `load bars.csv` can never reach `resample`; a scalar stored by `quant … as s` comes back from the saved environment as `json.Number`, so the first command that reads it will not find a `float64`; and `newdl 0.01 -0.004` fails in one-shot mode because Cobra reads `-0.004` as a flag. All three are small and were approved to be done together on 2026-09-05.

## What Changes

- **Core**: new `DataList.ParseDates(layouts ...string) *DataList` — every string cell is tried against the given Go layouts (default: the same list `ReadSQLOptions.ParseDates` uses) and becomes `time.Time` in UTC; cells that are already `time.Time` pass through; anything unparsable becomes `nil`. New `DataTable.ParseDatesCols(cols []string, layouts ...string) *DataTable` applies it in place to the named columns and is what `load sql … parsedates` already does internally, now shared. Added to `IDataList`/`IDataTable`.
- **CLI**: new `parsedates <var> [cols c1,c2] [layout <go-layout>] [as <var>]`. On a `DataList` it converts the whole list; on a `DataTable`, `cols` is required. `layout` may repeat. The result is stored under `as` or `$result`.
- **CLI env**: `LoadState` runs top-level scalar values through the same numeric coercion DataList elements get, so a saved `float64` reloads as `float64` and an `int64` as `int64`.
- **CLI**: `newdl`, `addrow`, and `addcol` set `DisableFlagParsing` so negative literals reach the command; `help <cmd>` still works because the registry routes `help` itself. Other value-taking commands are listed in the docs with the `--` workaround.
- `Docs/DataList.md`, `Docs/DataTable.md`, `Docs/cli-dsl.md`, `skills/insyra`, `skills/use-insyra-cli`, and both changelogs are updated; the three `AGENTS.md` Follow-ups entries are deleted.

## Capabilities

### New Capabilities

- `datalist-parse-dates`: string-to-`time.Time` conversion on `DataList`/`DataTable` and the `parsedates` CLI command.
- `cli-env-scalar-roundtrip`: scalar environment variables keep their numeric type across save and load.
- `cli-negative-literals`: `newdl`, `addrow`, `addcol` accept negative numbers in one-shot mode.

### Modified Capabilities

(none)

## Impact

- `datalist.go` or a new `datalist_dates.go` (+ test), `datatable_from_sql.go` (delegate `ParseDates` to the shared method), `interfaces.go`.
- New `cli/commands/parsedates.go` (+ test); `cli/commands/newdl.go`, `addrow.go`, `addcol.go` (+ a one-shot negative-literal test through `BuildCobraCommands`); `cli/env/state.go` (+ round-trip test).
- `Docs/DataList.md`, `Docs/DataTable.md`, `Docs/cli-dsl.md`, `skills/insyra/SKILL.md`, `skills/use-insyra-cli/SKILL.md` and references, `CHANGELOG.md`, `CHANGELOG_TW.md` (`### Core` and `### CLI`), `AGENTS.md`.
- No new dependencies, no breaking changes.
