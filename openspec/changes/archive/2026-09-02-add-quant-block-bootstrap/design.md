# Design: add-quant-block-bootstrap

## Context

`quant` takes raw series as `insyra.IDataList`, keeps computed series as `[]float64`, returns errors rather than logging, validates config structs instead of defaulting them (`WalkForwardConfig`), and pairs every exported function with an unexported `...F64` core for tests. `WalkForwardResult` already bundles a stitched return series with its compounded equity curve (from 1.0, length n+1). The core package's `quantileType7` is unexported; `DataList.Percentile` is the only exported route to it. `math/rand/v2` PCG is the seeded generator the core sampling code uses. ENG.md records that `DataList.ToF64Slice` has no failure channel and that new numeric analyses must not use it; `stats` converts through a validating helper built on the exported `insyra.ToFloat64Safe`.

## Goals / Non-Goals

**Goals:**
- One call yields both resampled returns and their equity paths, reproducible from a seed.
- Moving block and stationary bootstrap behind one config switch.
- Percentile bands that agree exactly with `DataList.Percentile`.
- Invalid input is refused, never silently coerced.

**Non-Goals:**
- Path statistics (loss probability, drawdown distribution, bootstrapped Sharpe). They are a few lines over `Returns`/`Equity`; adding them here widens the ticket without a requester.
- `DataTable` in or out for the path matrix. Thousands of columns each backed by an actor is the wrong container; the docs' input convention already says computed series stay `[]float64`.
- Defaults for `Horizon`/`BlockSize`/`Paths`, a minimum-sample refusal, an equity floor at 0, or rejecting `r < -1`. Statistical guidance goes in the docs; the API only refuses what it cannot read.
- CLI/DSL exposure; `quant` has no CLI surface today.
- Changing existing `quant` functions off `ToF64Slice` (standing follow-up in AGENTS.md).

## Decisions

### `BlockBootstrap` returns a result with both `Returns` and `Equity`

The issue sketched `BlockBootstrapPaths` returning equity paths. Equity is one consumer of the resampled returns; a bootstrapped Sharpe or per-path return statistic needs the returns themselves, and recovering them from equity (`E[t]/E[t-1] - 1`) is lossy. Returning only returns forces every fan-chart user to compound by hand. `WalkForwardResult` already resolves the same tension by carrying both, so `BootstrapResult{Returns, Equity}` follows the package's own precedent. Memory doubles (≈20 MB at 5000 × 252), which is acceptable. The name `BlockBootstrap` says what the function does — every statistics library's bootstrap returns resamples — instead of naming one derived product.

### Two methods behind `Stationary bool`

A boolean choosing between two named methods is usually better as an enum, but `WalkForwardConfig.Anchored` is the package's existing pattern for exactly this shape, and the zero value is the method the issue asked for. `BlockSize` means "fixed block length" for moving block and "mean block length" for stationary, which the doc comment states.

- **Moving block**: `k = n - BlockSize + 1` possible starts; draw `start` uniform in `[0, k)`, append `returns[start : start+BlockSize]`, repeat until `Horizon` values, truncate.
- **Stationary**: draw `start` uniform in `[0, n)`; draw block length `L = 1 + floor(ln U / ln(1-p))` with `p = 1/BlockSize` and `U` uniform in `(0, 1]` (`L = 1` when `BlockSize == 1`); append `returns[(start+j) mod n]` for `j < L`; repeat until `Horizon` values, truncate.

Both consume one PCG stream in path order, block order, so the output is a pure function of `(returns, cfg)`.

### The random stream depends only on PCG

`rand.New(rand.NewPCG(seed, seed ^ 0x9E3779B97F4A7C15))` mirrors the core sampling seeding. The uniform-integer and uniform-float reductions are implemented in `quant` from `Uint64()` (rejection sampling for integers, `(v>>11 + 1) * 2^-53` for `(0, 1]`) rather than through `rand.IntN`/`Float64`, so reproducibility rests on the PCG sequence alone — an algorithm with a published reference — and not on the standard library's reduction routines, whose stability across Go releases is not documented. A golden test with a fixed seed pins the output.

### Percentiles: a local type-7 copy, checked against the core

`quantileType7` is unexported and exporting a slice-level quantile from the root package is a core API decision outside this ticket. Building a `DataList` per time step and calling `Percentile` would sort the same column once per percentile (Horizon × len(percentiles) sorts of `Paths` values) and pay actor overhead each time. `quant` therefore carries a 20-line `quantileType7` over an already-sorted column and sorts each column once; a test asserts value-for-value agreement with `DataList.Percentile` on random data so the two cannot drift silently. Percentiles are taken on the `0..100` scale, matching `DataList.Percentile`.

### Input conversion refuses what it cannot read

A `quant`-local `numericSeries(dl, label)` reads under `AtomicDo`, converts with `insyra.ToFloat64Safe`, and returns an error naming the row for any non-numeric, NaN, or Inf value — the same shape as `stats.numericValues`. It is not shared with `stats` because neither package exports it and a cross-package helper is not this ticket's concern.

### Validation, not defaults

`Horizon <= 0`, `Paths <= 0`, `BlockSize < 1`, `BlockSize > len(returns)`, or an empty series returns an error naming the field, as `WalkForward` does. `PercentileBands` refuses an empty path set, ragged paths, an empty percentile list, or a percentile outside `[0, 100]`.

## Risks / Trade-offs

- [Local type-7 copy drifts from the core] → the agreement test against `DataList.Percentile` fails the build the moment they differ.
- [Callers assume `Seed: 0` means "random"] → doc comment and `Docs/quant.md` state that the seed always applies; identical runs are the audit property the issue asked for.
- [Stationary block lengths are unbounded in principle] → each draw is capped by `Horizon - len(series)` when appended, so a long draw simply fills the remainder; the geometric mean is still verified by test.
- [Twice the memory of an equity-only result] → documented; at the issue's scale it is tens of megabytes.

## Open Questions

None that block implementation.
