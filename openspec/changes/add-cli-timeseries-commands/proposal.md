# Proposal: add-cli-timeseries-commands

## Why

`add-timeseries-basics` (archived 2026-09-04) added `DataList.EWM`, `Rolling.Cov`/`Beta`, and `DataTable.Resample` to the Go API only. The CLI's `rolling` command still stops at `corr`, there is no `ewm`, and the only way to turn daily bars into monthly bars from a `.isr` script is a hand-written `groupby`. Research staff who work through the REPL rather than Go cannot reach the new primitives at all.

## What Changes

- New `ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>] [as <var>]`. Exactly one decay keyword, mirroring `EWMOptions`; the result is a same-length `DataList` stored under `as` or `$result` like `rolling`.
- `rolling` gains two reducers that take a second `DataList` argument: `rolling <var> <window> cov <other> [...]` and `rolling <var> <window> beta <other> [...]`. Existing reducers and options are unchanged; `help rolling` lists the new forms.
- New `resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] [<col>:<op>[:<name>] ...] [as <var>]`. `op` is one of the `groupby` operator names (`sum mean median min max count first last std var`); the optional third field renames the output column. Result is a `DataTable`.
- Errors from the library (bad decay, non-time cells, unknown op) surface as command errors with the library's message; nothing is defaulted silently.
- `Docs/cli-dsl.md` (Command Groups "Time Series / Transforms", Full Command Index rows, a resample example under the quickstart workflows), `skills/use-insyra-cli` (`SKILL.md` and `references/cli-commands.md`, `cli-command-guide.md`, `cli-command-usage.md`), and both changelogs (`### CLI`) are updated in the same change.

## Capabilities

### New Capabilities

- `cli-timeseries-commands`: `ewm`, `rolling cov|beta`, and `resample` in the CLI/REPL/DSL.

### Modified Capabilities

(none)

## Impact

- `cli/commands/timeseries.go` (+ `ewm`, `rolling cov|beta`), new `cli/commands/resample.go`, tests in `cli/commands/timeseries_test.go` (new) and `cli/commands/resample_test.go` (new).
- `Docs/cli-dsl.md`, `skills/use-insyra-cli/SKILL.md` and `references/*.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No library changes, no new dependencies.
