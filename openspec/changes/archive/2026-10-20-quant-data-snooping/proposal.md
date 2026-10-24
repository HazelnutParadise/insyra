# Proposal: quant-data-snooping

## Why

[#478](https://github.com/HazelnutParadise/insyra/issues/478), part 3: after trying many strategies or parameter sets and keeping the best one, a researcher needs to know whether the winner really beats the benchmark or only won by luck. `DeflatedSharpeRatio` and `PBO` look at the spread of Sharpe ratios and the stability of rankings, and `stats.AdjustPValues` ignores how correlated the strategies are. White's Reality Check and Hansen's test for Superior Predictive Ability answer the question directly with a bootstrap that keeps that correlation.

## What Changes

- `quant.RealityCheck(perf, opts)` and `quant.SPATest(perf, opts)` take the same T×N performance table as `PBO` and return the p-value of "the best column does not beat the benchmark" (SPA: the lower, consistent and upper p-values).
- `DataSnoopingOptions` holds the benchmark (nil compares with 0), the number of bootstrap replications, the block size, the seed and the resampling scheme (`ResamplingScheme`: stationary by default, or moving block).
- Every replication draws one set of time indices and applies it to every column, so the correlation between strategies is kept.
- `BlockBootstrap`'s resampling is split into a step that draws indices and a step that reads values, so both features share it. `BlockBootstrap`'s output for a given seed does not change.
- Romano-Wolf stepdown is not part of this change.

## Capabilities

### New Capabilities

- `quant-data-snooping`: White's Reality Check and Hansen's SPA test for the best of many strategies.

## Impact

`quant/data_snooping.go`, `quant/bootstrap.go` (refactor only), `quant/enum_names.go`, tests and a Python reference in `quant/testdata`, `Docs/quant.md`, both CHANGELOGs.
