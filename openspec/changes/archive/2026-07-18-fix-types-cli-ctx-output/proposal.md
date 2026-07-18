# Route the `types` CLI command through ctx.Output

## Why

The CLI convention (cli/AGENTS.md) is that every command writes through `ctx.Output`, which the REPL and `run` capture. M20 fixed this for `show`/`describe`/`summary` by adding writer-aware `ShowTo`/`ShowRangeTo`/`SummaryTo` methods, but `ShowTypes`/`ShowTypesRange` were intentionally left stdout-only at the time. The `types` command (`cli/commands/types.go`) therefore still prints straight to `os.Stdout`: its output cannot be captured by the REPL output path, asserted in tests, or redirected. Tracked as the `[2026-07-10]` Follow-ups entry in `AGENTS.md`.

## What Changes

- Add writer-aware variants on both core types, mirroring the M20 pattern exactly:
  - `(*DataTable) ShowTypesTo(w io.Writer)` and `(*DataTable) ShowTypesRangeTo(w io.Writer, startEnd ...any)`
  - `(*DataList) ShowTypesTo(w io.Writer)` and `(*DataList) ShowTypesRangeTo(w io.Writer, startEnd ...any)`
  - The existing `ShowTypes`/`ShowTypesRange` delegate to the `To` variants with `os.Stdout`; rendered output is byte-identical.
  - Thread `w` through the shared helper `printTypeRows` (same as `printRowsColored`).
- `cli/commands/types.go` calls `ShowTypesTo(ctx.Output)` for both DataTable and DataList.
- Regression test in `cli/commands` proving `types` output reaches `ctx.Output` (same style as `m20_output_test.go`).
- Docs: add the `To` variants to the `ShowTypes`/`ShowTypesRange` sections of `Docs/DataTable.md` and `Docs/DataList.md` (same inline style as `ShowTo`/`ShowRangeTo`).
- Delete the resolved `[2026-07-10]` `types` Follow-ups entry from `AGENTS.md`.

## Capabilities

### Modified Capabilities
- `cli-output-capture`: all built-in CLI commands, now including `types`, emit their rendered output through `ctx.Output` so REPL/`run`/tests can capture it.

## Impact

- **Code**: `show.go` (4 new exported methods, `printTypeRows` gains a writer param), `cli/commands/types.go`, new `cli/commands/types_test.go`.
- **Docs**: `Docs/DataTable.md`, `Docs/DataList.md` (signature blocks + one-line description mention). `Docs/cli-dsl.md` unchanged (command usage/behavior for interactive users is identical). `skills/` unchanged (no skill references `ShowTypes`).
- **API surface**: additive only. Following the M20 precedent, the `To` variants are NOT added to `IDataList`/`IDataTable` in `interfaces.go`.
- **Behavior**: interactive terminal output is unchanged (`ShowTypes()` still writes to stdout). The only observable change is that `types` output is now captured wherever `ctx.Output` is captured.
