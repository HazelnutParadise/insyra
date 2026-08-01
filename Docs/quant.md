# [ quant ] Package

This document describes all public APIs in the `quant` package, designed for AI/automated applications to directly understand each function, type, parameter, and return value.

---

## Installation

```bash
go get github.com/HazelnutParadise/insyra/quant
```

---

## Overview

The `quant` package provides quantitative-finance tools for evaluating trading strategies and portfolios:

- **Performance metrics**: `SharpeRatio`, `MaxDrawdown`, `AnnualizedReturn` — headline risk/return numbers from a return series or equity curve
- **Backtest-overfitting diagnostics**: `ProbabilisticSharpeRatio`, `ExpectedMaxSharpe`, `DeflatedSharpeRatio`, `PBO` — quantify how much of a backtest's edge is real versus selection bias from multiple testing (Bailey & López de Prado)
- **Walk-forward validation**: `WalkForward` — slide train/test windows, pick parameters in-sample, evaluate out-of-sample, and stitch the out-of-sample track record together (Pardo)

Unlike the [`finance`](./finance.md) package (which uses high-precision `decimal.Decimal` for TVM, NPV/IRR, and bond pricing), `quant` works with ordinary floating-point numbers — the industry convention for return/equity analytics, where statistical noise dwarfs floating-point error.

**Input convention.** Functions that take your *raw data series* accept `insyra.IDataList` (a return or equity column) or `insyra.IDataTable` (a strategy×period matrix), the same as the [`stats`](./stats.md) package. Values that are *not* raw data — scalar Sharpe/variance inputs, and the returns/equity that walk-forward *computes* — stay as `float64`. Every exported function follows an **error-first** convention: invalid input returns an `error` rather than logging or panicking. Always handle `err` at the call site.

> **Annualized vs per-period Sharpe.** `SharpeRatio` returns an *annualized* Sharpe (it multiplies by `√periodsPerYear`). The overfitting diagnostics (`ProbabilisticSharpeRatio`, `DeflatedSharpeRatio`, and the Sharpe ratios you feed to them) use the *per-period, non-annualized* Sharpe — i.e. `mean/stddev` with no annualization. Compute that with `SharpeRatio(returns, 0, 1)`. Mixing the two conventions silently corrupts DSR/PBO results.

---

## Performance Metrics

### SharpeRatio

```go
func SharpeRatio(returns insyra.IDataList, riskFreeRate, periodsPerYear float64) (float64, error)
```

Annualized Sharpe ratio of a periodic return series:

```text
Sharpe = mean(returns - riskFreeRate) / stddev(returns) · √periodsPerYear
```

**Parameters:**

- `returns`: per-period simple returns (e.g. daily returns as `0.012` for +1.2%)
- `riskFreeRate`: risk-free rate expressed in the **same period** as `returns` (pass `0` for an excess-return series or to ignore it)
- `periodsPerYear`: annualization factor — `252` for daily Taiwan-stock returns, `52` weekly, `12` monthly

The standard deviation is the **sample (n-1)** standard deviation, matching the common backtesting convention. Pass `periodsPerYear = 1` to obtain the per-period (non-annualized) Sharpe used by the overfitting diagnostics.

**Returns:** `(sharpe, err)` — `err` is non-nil for fewer than 2 returns, non-positive `periodsPerYear`, or a zero-volatility series (Sharpe undefined).

### MaxDrawdown

```go
func MaxDrawdown(equity insyra.IDataList) (float64, error)
```

Maximum drawdown of an equity (cumulative value / NAV) curve, returned as a **non-negative fraction**: `0.2` means the curve fell 20% below a prior running peak at its worst point. A monotonically non-decreasing curve returns `0`.

`equity` should be a positive value series; points where the running peak is non-positive are skipped (drawdown is undefined there).

**Returns:** `(drawdown, err)` — `err` is non-nil only for an empty `equity`.

### AnnualizedReturn

```go
func AnnualizedReturn(equity insyra.IDataList, days int) (float64, error)
```

Annualized (CAGR-style) return implied by an equity curve spanning `days` **calendar** days:

```text
(equity[last] / equity[0]) ^ (365 / days) - 1
```

Only the first and last points of `equity` matter; `days` is the calendar-day span the curve covers.

**Returns:** `(annualized, err)` — `err` is non-nil for fewer than 2 points, non-positive `days`, or a non-positive first/last value.

---

## Backtest-Overfitting Diagnostics

These implement the framework of Bailey, Borwein, López de Prado & Zhu. All Sharpe ratios here are **per-period, non-annualized**.

### ProbabilisticSharpeRatio

```go
func ProbabilisticSharpeRatio(observedSR, benchmarkSR float64, n int, skew, kurt float64) (float64, error)
```

The Probabilistic Sharpe Ratio (PSR): the probability that the true Sharpe exceeds a benchmark, given the estimate's standard error under non-normal returns.

```text
PSR = Φ[ (SR̂ - SR*)·√(n-1) / √(1 - γ₃·SR̂ + ((γ₄-1)/4)·SR̂²) ]
```

**Parameters:**

- `observedSR` (SR̂), `benchmarkSR` (SR*): per-period, non-annualized Sharpe ratios (scalars)
- `n`: number of return observations
- `skew` (γ₃): skewness of the returns
- `kurt` (γ₄): **non-excess** kurtosis of the returns (a normal distribution has `skew = 0`, `kurt = 3`)

**Returns:** `(psr, err)` — `err` is non-nil for `n < 2` or a non-positive variance term in the denominator (possible under extreme skew/kurtosis). `Φ` is the standard-normal CDF ([`stats.NormCDF`](./stats.md)).

### ExpectedMaxSharpe

```go
func ExpectedMaxSharpe(sharpeVariance float64, nTrials int) (float64, error)
```

SR₀, the expected **maximum** per-period Sharpe obtained by chance after `nTrials` independent backtests whose Sharpe ratios have variance `sharpeVariance`. This is the deflation benchmark used by `DeflatedSharpeRatio`:

```text
SR₀ = √V · [ (1-γ)·Z⁻¹(1 - 1/N) + γ·Z⁻¹(1 - 1/(N·e)) ]
```

where `V = sharpeVariance`, `N = nTrials`, `γ` is the Euler-Mascheroni constant, `e` is Euler's number, and `Z⁻¹` is the standard-normal quantile ([`stats.NormPPF`](./stats.md)).

With `nTrials ≤ 1` there is no selection bias, so `SR₀ = 0`.

**Returns:** `(sr0, err)` — `err` is non-nil for negative `sharpeVariance`.

### DeflatedSharpeRatio

```go
func DeflatedSharpeRatio(observedSR float64, n int, skew, kurt float64, trialSharpes insyra.IDataList) (float64, error)
```

The Deflated Sharpe Ratio (DSR): the PSR of the selected strategy measured against the deflation benchmark SR₀ derived from the whole set of trial Sharpe ratios. It corrects the observed Sharpe for **selection bias from multiple testing, non-normality, and sample length** in one number. **DSR ≈ 1** means the result survives deflation; **DSR near 0** means it is likely a false discovery.

**Parameters:**

- `observedSR`: the selected strategy's per-period (non-annualized) Sharpe, typically the maximum of `trialSharpes`
- `n`: its number of return observations
- `skew`, `kurt`: that strategy's skewness and non-excess kurtosis
- `trialSharpes`: an `IDataList` of the per-period Sharpe ratios of **all** trials considered during the search; their count and (population) variance feed SR₀

Equivalent to `ProbabilisticSharpeRatio(observedSR, ExpectedMaxSharpe(var(trialSharpes), trialSharpes.Len()), n, skew, kurt)`.

**Returns:** `(dsr, err)` — `err` is non-nil for an empty `trialSharpes` or any downstream failure.

### PBO

```go
func PBO(perf insyra.IDataTable, nSplits int) (float64, error)
```

Estimates the **Probability of Backtest Overfitting** via Combinatorially Symmetric Cross-Validation (CSCV).

**Parameters:**

- `perf`: a `T×N` performance `DataTable` — **column j is candidate strategy j, row i is period i**, so `perf[i][j]` is strategy j's period-i return (`T` time rows, `N` strategies, `N ≥ 2`). All columns must have equal length.
- `nSplits` (S): the number of equal, contiguous time blocks the rows are cut into; must be a positive **even** number and `≤ T`

CSCV enumerates every way to split the `S` blocks into an in-sample half (IS) and an out-of-sample half (OOS). For each split it picks the IS-best strategy by Sharpe ratio and records its OOS rank; PBO is the fraction of splits where that strategy's OOS performance lands in the bottom half (logit ω ≤ 0). **A high PBO means in-sample winners tend to be out-of-sample losers** — the signature of overfitting.

If `T` is not a multiple of `nSplits`, the trailing `T mod nSplits` rows are dropped. Per-block Sharpe uses the sample standard deviation; a zero-volatility series contributes a Sharpe of 0.

**Returns:** `(pbo, err)` — a probability in `[0, 1]`. `err` is non-nil for an empty matrix, columns of unequal length, fewer than 2 strategies, an odd or non-positive `nSplits`, or `nSplits > T`.

> Combination count is `C(S, S/2)`, which grows fast: `S=16 → 12,870`. Keep `nSplits` modest (typically 8–16).

---

## Walk-Forward Validation

### Types

```go
type WalkForwardConfig struct {
    TrainSize int  // in-sample periods used to pick parameters per fold
    TestSize  int  // out-of-sample periods evaluated per fold; folds advance by TestSize
    Anchored  bool // false: fixed-size rolling train window; true: expanding window anchored at 0
}

type WalkForwardFold struct {
    TrainStart int // all ranges are half-open [Start, End)
    TrainEnd   int
    TestStart  int
    TestEnd    int
    OOSReturns []float64 // per-period out-of-sample returns for this fold
}

type WalkForwardResult struct {
    Folds      []WalkForwardFold
    OOSReturns []float64 // stitched out-of-sample returns, chronological
    Equity     []float64 // compounded OOS equity curve starting at 1.0 (len == len(OOSReturns)+1)
}
```

### WalkForward

```go
func WalkForward[P any](
    n int,
    cfg WalkForwardConfig,
    optimize func(trainStart, trainEnd int) P,
    evaluate func(p P, testStart, testEnd int) []float64,
) (*WalkForwardResult, error)
```

Runs a time-series walk-forward (out-of-sample) validation over `n` periods. For each fold it calls `optimize` on the in-sample window to pick parameters of type `P`, then `evaluate` on the out-of-sample window to obtain that fold's per-period returns; the out-of-sample returns are stitched together and compounded into a single equity curve. This is the standard guard against optimizing and evaluating on the same data.

This function is intentionally **index-driven** rather than `IDataList`-based: both callbacks receive half-open `[start, end)` index ranges into **your own** data (close over the actual series), so any data layout and any parameter type `P` work. `evaluate` should return one return per out-of-sample period (typically `testEnd - testStart` values).

Windows advance by `TestSize` starting at `TrainSize`. With a rolling window, fold k is train `[TestStart-TrainSize, TestStart)`, test `[TestStart, TestStart+TestSize)`; `Anchored` fixes the training start at 0 (expanding). If `n-TrainSize` is not a multiple of `TestSize`, the final out-of-sample window is shorter than `TestSize` rather than dropped, so all data is used.

**Returns:** `(*WalkForwardResult, err)` — `err` is non-nil for non-positive `n`/`TrainSize`/`TestSize`, `TrainSize >= n` (no room to test), or a nil callback.

### Result helpers

```go
func (r *WalkForwardResult) Sharpe(riskFreeRate, periodsPerYear float64) (float64, error)
func (r *WalkForwardResult) MaxDrawdown() (float64, error)
func (r *WalkForwardResult) AnnualizedReturn(days int) (float64, error)
```

Convenience aggregations over the stitched out-of-sample track record — they apply the same Sharpe/drawdown/CAGR formulas as the package functions to `r.OOSReturns` / `r.Equity` directly.

---

## Usage Examples

### Headline performance metrics

```go
package main

import (
    "fmt"

    "github.com/HazelnutParadise/insyra"
    "github.com/HazelnutParadise/insyra/quant"
)

func main() {
    returns := insyra.NewDataList(0.012, -0.004, 0.008, 0.0, 0.015, -0.009)
    equity  := insyra.NewDataList(100.0, 101.2, 100.8, 101.6, 101.6, 103.1, 102.2)

    sharpe, _ := quant.SharpeRatio(returns, 0, 252) // annualized, daily
    mdd, _    := quant.MaxDrawdown(equity)
    cagr, _   := quant.AnnualizedReturn(equity, 30) // 30 calendar days

    fmt.Printf("Sharpe=%.3f  MaxDD=%.2f%%  CAGR=%.2f%%\n", sharpe, mdd*100, cagr*100)
}
```

Already have a column in a `DataTable`? Pass it straight in: `quant.SharpeRatio(dt.GetCol("returns"), 0, 252)`.

### Deflated Sharpe Ratio after a parameter search

```go
// Suppose you tried 40 configurations and recorded each one's per-period
// (non-annualized) Sharpe in an IDataList. The best config is `best`.
trialSharpes := collectPerPeriodSharpes() // insyra.IDataList, length 40
best := bestReturns                       // insyra.IDataList, the selected config's returns

observedSR, _ := quant.SharpeRatio(best, 0, 1)   // per-period (periodsPerYear=1)
skew, kurt := momentsOf(best)                    // skewness, non-excess kurtosis

dsr, err := quant.DeflatedSharpeRatio(observedSR, best.Len(), skew, kurt, trialSharpes)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("DSR = %.3f (>0.95 ≈ survives deflation)\n", dsr)
```

### Probability of Backtest Overfitting

```go
// perf: a DataTable whose column j is strategy j and row i is period i.
pbo, err := quant.PBO(perf, 16)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("PBO = %.1f%%\n", pbo*100) // high → in-sample winners fail out-of-sample
```

### Walk-forward validation

```go
prices := loadDailyReturns() // []float64, length n
n := len(prices)

res, err := quant.WalkForward(n,
    quant.WalkForwardConfig{TrainSize: 252, TestSize: 63}, // ~1y train, ~1q test, rolling
    func(trainStart, trainEnd int) int {
        // Pick a parameter (e.g. a lookback) on the in-sample slice.
        return bestLookback(prices[trainStart:trainEnd])
    },
    func(lookback, testStart, testEnd int) []float64 {
        // Apply it out-of-sample; return one return per OOS period.
        return runStrategy(prices[testStart:testEnd], lookback)
    },
)
if err != nil {
    log.Fatal(err)
}

oosSharpe, _ := res.Sharpe(0, 252)
oosMDD, _    := res.MaxDrawdown()
fmt.Printf("OOS Sharpe=%.3f  OOS MaxDD=%.2f%%  folds=%d\n",
    oosSharpe, oosMDD*100, len(res.Folds))
```

---

## Error Handling

All exported functions return `(value, error)` and surface validation problems through the second return value. Common error sources:

- **`SharpeRatio`** — fewer than 2 returns, non-positive `periodsPerYear`, zero volatility
- **`MaxDrawdown`** — empty `equity`
- **`AnnualizedReturn`** — fewer than 2 points, non-positive `days`, non-positive first/last value
- **`ProbabilisticSharpeRatio`** — `n < 2`, non-positive variance term (extreme skew/kurtosis)
- **`ExpectedMaxSharpe`** — negative `sharpeVariance`
- **`DeflatedSharpeRatio`** — empty `trialSharpes`, or any downstream error
- **`PBO`** — empty matrix, columns of unequal length, fewer than 2 strategies, odd/non-positive `nSplits`, `nSplits > T`
- **`WalkForward`** — non-positive `n`/`TrainSize`/`TestSize`, `TrainSize >= n`, nil callback

The package never logs warnings on its own and never panics from valid input — it follows the same error-first contract as [`stats`](./stats.md) and [`finance`](./finance.md).

---

## Related Packages

- [`stats`](./stats.md): `NormCDF` / `NormPPF` (used internally by the overfitting diagnostics), plus skewness/kurtosis, hypothesis tests, and regression
- [`finance`](./finance.md): high-precision TVM, NPV/IRR, bonds — use it for exact cashflow/loan math rather than return-series analytics
- [`insyra`](../README.md): `DataList` / `DataTable` core types — the input types for the performance and overfitting functions. Build a `DataList` from raw numbers with `insyra.NewDataList(vals...)`.
