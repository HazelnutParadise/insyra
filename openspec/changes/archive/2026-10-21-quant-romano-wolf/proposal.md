# Proposal: quant-romano-wolf

## Why

[#478 策略檢定補齊](https://github.com/HazelnutParadise/insyra/issues/478), part 3b: `RealityCheck` and `SPATest` (part 3, 資料窺探檢定) say whether the best of many strategies beats a benchmark, but not which of the others also do. Romano and Wolf's stepdown names every strategy that beats the benchmark while aiming to hold the chance of even one false claim (the familywise error rate) at a chosen level; the control is asymptotic.

## What Changes

- `quant.RomanoWolf(perf, opts)` takes the same T×N table as `RealityCheck` and returns, for every column, the step at which it was declared to beat the benchmark (0 when it was not), its studentized statistic and the critical value of each step.
- `RomanoWolfOptions` embeds `DataSnoopingOptions` (benchmark, replications, block size, seed, scheme) and adds `Alpha`, the familywise level, which must be set.
- The test is one-sided (outperformance), as `RealityCheck` and `SPATest` are. It gives no adjusted p-values.
- Each replication re-estimates every column's long-run standard deviation on the resampled series with Hansen's exact kernel (the one `SPATest` uses), computing the autocovariances by FFT in O(T log T) per column and replication instead of O(T²).
- `SPATest` and `RealityCheck` numbers do not change.

## Capabilities

### New Capabilities

- `quant-romano-wolf`: the Romano-Wolf stepdown over many strategies at a familywise level.

## Impact

`quant/romano_wolf.go`, `quant/data_snooping.go` (comment only), tests and a Python reference in `quant/testdata`, `Docs/quant.md`, both CHANGELOGs.
