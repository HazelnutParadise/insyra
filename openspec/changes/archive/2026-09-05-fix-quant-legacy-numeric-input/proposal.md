# Proposal: fix-quant-legacy-numeric-input

## Why

`quant` now has two input behaviours. Everything added since `BlockBootstrap` (`CAPM`, `Beta`, the risk metrics, `FactorModel`) reads through `numericSeries` and refuses a cell it cannot read, naming the row. The four older entry points — `SharpeRatio`, `MaxDrawdown`, `AnnualizedReturn`, `DeflatedSharpeRatio`, and the column loop in `PBO` — still call `DataList.ToF64Slice`, which has no failure channel and silently turns a blank, a string, or a NaN into `0`. A blank in a return series therefore lowers the Sharpe ratio and flattens a drawdown without any signal; the standing AGENTS.md follow-up records that `stats` was moved off this path after a single blank moved a Pearson coefficient from 0.9992 to 0.9879. With the package now half-migrated, a user who gets an error from `SortinoRatio` and a number from `SharpeRatio` on the same series has no way to know the second one is wrong.

## What Changes

- `SharpeRatio`, `MaxDrawdown`, `AnnualizedReturn`, `DeflatedSharpeRatio` (its `trialSharpes` argument), and `PBO` (every column) read input through `numericSeries` and return an error naming the series and the one-based row for a non-numeric, `NaN`, or `Inf` cell. Labels: `returns`, `equity`, `trialSharpes`, and `column <j>` for `PBO`.
- Nothing else about these functions changes: signatures, numeric results on fully numeric input, existing validation messages, and the `...F64` cores are untouched. Existing tests must pass unchanged; new tests add the refusal cases.
- `WalkForwardResult.Sharpe/MaxDrawdown/AnnualizedReturn` are unaffected — they already hold `[]float64`.
- **BREAKING (behavioural)**: input that used to be silently coerced to `0` now returns an error. Marked in both changelogs the way past release notes mark breaking changes.
- The AGENTS.md follow-up "`ToF64Slice` still fabricates zeros for 54 callers outside `stats`" is updated: the `quant` callers are removed from its list; the remaining display-path callers stay as recorded.
- Docs (`Docs/quant.md` error-handling rows and the "Input convention" paragraph), the `skills/insyra` quant note, and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `quant-performance-input`: there is no existing spec for the legacy functions, so this change adds a delta spec under this new capability name that states the input contract for `SharpeRatio`, `MaxDrawdown`, `AnnualizedReturn`, `DeflatedSharpeRatio`, and `PBO`. It is listed here rather than under "New" because the functions already exist; the archive step creates `openspec/specs/quant-performance-input/spec.md`.

## Impact

- `quant/performance.go`, `quant/overfitting.go` (five call sites), `quant/performance_test.go`, `quant/overfitting_test.go` (refusal cases).
- `Docs/quant.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `AGENTS.md` (follow-up entry).
- No new dependencies. `ToF64Slice` itself is not changed; callers in `plot`, `gplot`, `cli`, and interpolation stay as the follow-up records.
