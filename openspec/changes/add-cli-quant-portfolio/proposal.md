# Proposal: add-cli-quant-portfolio

## Why

`add-quant-portfolio-optimization` landed after `add-cli-quant-commands`, so the `quant` command group reaches every `quant` function except the one that produces a decision: portfolio weights. A `.isr` script can compute Sharpe, VaR, beta, and option prices but cannot ask for a minimum-variance or maximum-Sharpe allocation.

## What Changes

- Two new forms in the existing `quant` command:
  - `quant portfolio <returns_dt> minvar|target <r>|maxsharpe [rf <r>] [min <v1,v2,…>] [max <v1,v2,…>] [as <var>]` → `OptimizePortfolio`. `<returns_dt>` is a `DataTable` of aligned per-period returns, one column per asset. `min`/`max` are comma-separated per-asset bounds in column order (default long-only `[0, 1]`); `rf` is per period (default 0). Prints one `asset=weight` line per asset and a summary line (`return=… vol=… sharpe=… iterations=… converged=…`). Stores a `DataTable` with columns `Asset, Weight` under `as` or `$result`, and a one-row `<var>_stats` table with `ExpectedReturn, Variance, Volatility, SharpeRatio, Iterations, Converged`.
  - `quant frontier <returns_dt> <points> [rf <r>] [min …] [max …] [as <var>]` → `EfficientFrontier`. Stores a `DataTable` with one row per point: `ExpectedReturn, Variance, Volatility, SharpeRatio, Converged`, followed by one weight column per asset named after the asset.
- A non-converged solve is reported in the printed summary and the `Converged` column; it is not an error, matching the library.
- Library errors surface with the `quant portfolio:` / `quant frontier:` prefix; a bounds list of the wrong length or a non-numeric bound is rejected before the call.
- `Docs/cli-dsl.md` (`quant` Forms, Full Command Index row, a "Portfolio weights from the CLI" quickstart), `skills/use-insyra-cli` (`SKILL.md` and references), and both changelogs (`### CLI`) are updated in the same change.

## Capabilities

### New Capabilities

- `cli-quant-portfolio`: the `quant portfolio` and `quant frontier` forms.

### Modified Capabilities

(none — existing `quant` forms are unchanged)

## Impact

- `cli/commands/quant.go` (two entries in `quantForms` and their run functions), `cli/commands/quant_test.go` (or a new `quant_portfolio_test.go`).
- `Docs/cli-dsl.md`, `skills/use-insyra-cli/SKILL.md` and `references/*.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No library changes, no new dependencies.
