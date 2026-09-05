# Proposal: add-cli-quant-commands

## Why

`quant` now covers performance, risk, exposure, factor attribution, and option pricing, but none of it is reachable from the CLI, REPL, or `.isr` scripts — the surfaces a research or risk desk actually types into. `stats` has `ttest`, `regression`, `pca`; `quant` has nothing. A `quant` command group with one form per function closes that gap without inventing new semantics: every argument maps to a Go parameter, and every number printed is the Go result.

## What Changes

- New `quant` command with sub-forms (all series arguments are `DataList` variables; scalars are numbers; `[as <var>]` stores the result):
  - `quant sharpe <returns> <periods> [rf <r>]`, `quant sortino <returns> <periods> [mar <r>]`, `quant ir <returns> <benchmark> <periods>`
  - `quant maxdd <equity>`, `quant annret <equity> <days>`, `quant calmar <equity> <days>`, `quant drawdown <equity> [as <var>]` (DataList)
  - `quant var <returns> <confidence> [historical|parametric]`, `quant cvar <returns> <confidence> [historical|parametric]` (default historical)
  - `quant beta <asset> <market>`, `quant capm <asset> <market> [rf <r>] [as <var>]` (prints beta, alpha, R², standard errors, N; stores a one-row `DataTable`)
  - `quant factor <asset> <factors_dt> [rf <r>] [as <var>]` (prints alpha and one line per factor; stores a `DataTable` with a row per factor: `Factor, Exposure, StdErr, TValue, PValue`)
  - `quant bs call|put <spot> <strike> <rate> <vol> <years> [q <yield>] [as <var>]` (prints price and greeks; stores a one-row `DataTable`)
  - `quant iv call|put <price> <spot> <strike> <rate> <years> [q <yield>] [as <var>]`
- Scalar results are printed as `name=value` and, when `as` is given, stored as a `float64` variable, matching `corr`. `periods`, `days`, `confidence` are required positionals because the library refuses to default them.
- Library errors surface unchanged with a `quant <form>:` prefix.
- `Docs/cli-dsl.md` (Command Groups "Modeling / Viz / Fetch" → add a "Quant" line, Full Command Index row, a "Risk report from a return series" quickstart), `skills/use-insyra-cli` (`SKILL.md` and references), and both changelogs (`### CLI`) are updated in the same change.

## Capabilities

### New Capabilities

- `cli-quant-commands`: the `quant` command group exposing performance, risk, exposure, factor, and option functions.

### Modified Capabilities

(none)

## Impact

- New `cli/commands/quant.go`, `cli/commands/quant_test.go`.
- `Docs/cli-dsl.md`, `skills/use-insyra-cli/SKILL.md` and `references/*.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- `cli/commands` gains an import of `quant`; no new external dependencies.
