# Proposal: add-quant-block-bootstrap

## Why

`quant` can evaluate a strategy's past (`SharpeRatio`, `MaxDrawdown`, `PBO`, `WalkForward`) but offers nothing for the question users actually ask next: "if I keep this configuration, where might I be in a year?" A point forecast of the return is unreliable and misleading; a *distribution* — median with a confidence band, probability of loss, drawdown spread — is defensible and states its uncertainty honestly. Block bootstrap gives that distribution without assuming normality, keeping the autocorrelation, volatility clustering, and fat tails of the observed return series, and is reproducible given a seed. Callers currently reimplement it locally ([issue #199](https://github.com/HazelnutParadise/insyra/issues/199), requested while building easy-invest).

## What Changes

- New `quant.BootstrapConfig{Horizon, BlockSize, Paths, Seed, Stationary}` and `quant.BlockBootstrap(returns insyra.IDataList, cfg) (*BootstrapResult, error)`. The result carries both `Returns` (Paths × Horizon resampled per-period returns) and `Equity` (Paths × (Horizon+1) compounded from 1.0, the same convention as `WalkForwardResult.Equity`), so a fan chart uses `Equity` and a bootstrapped Sharpe distribution uses `Returns` from the same call.
- `Stationary: false` (default) is the moving block bootstrap (Künsch 1989): fixed-length blocks, start uniform in `[0, n-BlockSize]`. `Stationary: true` is the stationary bootstrap (Politis & Romano 1994): geometric block lengths with mean `BlockSize`, circular indexing.
- New `quant.PercentileBands(paths [][]float64, percentiles []float64) ([][]float64, error)`: for every time step, the requested percentiles across all paths, using the R type-7 quantile that `DataList.Percentile`, `Quartile`, and `Describe` already share. `bands[i]` is the series for `percentiles[i]`, in the caller's order.
- Input conversion refuses non-numeric, NaN, and Inf values with an error naming the row, the way `stats` does. The new code does not use `DataList.ToF64Slice` (see ENG.md: it fabricates zeros). Existing `quant` functions are left as they are.
- Configuration is validated, not defaulted, following `WalkForwardConfig`: `Horizon > 0`, `Paths > 0`, `1 <= BlockSize <= len(returns)`.
- Same seed and inputs produce bit-identical output; the random stream is a `math/rand/v2` PCG seeded from `Seed` alone, with the uniform-integer reduction done in `quant` so results depend only on the PCG sequence.
- Docs (`Docs/quant.md`, `Docs/README.md` and both README package rows), skills (`skills/insyra`), and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

- `quant-bootstrap`: block-bootstrap resampling of a return series into simulated return and equity paths, and percentile bands over a path matrix.

### Modified Capabilities

(none — no existing spec-level requirement changes)

## Impact

- New files `quant/bootstrap.go`, `quant/bootstrap_test.go`; a small numeric-input helper inside `quant`.
- `Docs/quant.md` (new "Probabilistic forecasting" section, overview bullet, usage example, related-packages note), `Docs/README.md`, `README.md`, `README_TW.md` (quant row descriptions), `skills/insyra/SKILL.md` (new quant section), `CHANGELOG.md`, `CHANGELOG_TW.md`.
- No breaking changes, no new dependencies, no CLI/DSL surface (quant has none today).
