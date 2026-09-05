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
- **Risk metrics**: `ValueAtRisk`, `ConditionalValueAtRisk`, `SortinoRatio`, `CalmarRatio`, `InformationRatio`, `DrawdownSeries` — tail risk, downside performance, benchmark-relative performance, and per-period drawdowns
- **Market exposure**: `Beta`, `CAPM` — measure an asset's exposure and per-period alpha against a benchmark return series
- **Options**: `BlackScholes`, `ImpliedVolatility` — price European calls and puts, report five greeks, and recover volatility from a market price
- **Factor models**: `FactorModel` — attribute an asset's excess returns to named market, size, value, momentum, or other factor columns
- **Backtest-overfitting diagnostics**: `ProbabilisticSharpeRatio`, `ExpectedMaxSharpe`, `DeflatedSharpeRatio`, `PBO` — quantify how much of a backtest's edge is real versus selection bias from multiple testing (Bailey & López de Prado)
- **Walk-forward validation**: `WalkForward` — slide train/test windows, pick parameters in-sample, evaluate out-of-sample, and stitch the out-of-sample track record together (Pardo)
- **Probabilistic forecasting**: `BlockBootstrap`, `PercentileBands` — resample a return series in blocks (moving block or stationary bootstrap) into thousands of simulated equity paths and take percentile bands for a fan chart, reproducible from a seed

Unlike the [`finance`](./finance.md) package (which uses high-precision `decimal.Decimal` for TVM, NPV/IRR, and bond pricing), `quant` works with ordinary floating-point numbers — the industry convention for return/equity analytics, where statistical noise dwarfs floating-point error.

**Input convention.** Functions that take your *raw data series* accept `insyra.IDataList` (a return or equity column) or `insyra.IDataTable` (a strategy×period matrix), the same as the [`stats`](./stats.md) package. Values that are *not* raw data — scalar Sharpe/variance inputs, and the returns/equity that walk-forward and the bootstrap *compute* — stay as `float64`. Every exported function follows an **error-first** convention: invalid input returns an `error` rather than logging or panicking. Always handle `err` at the call site.

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

## Risk Metrics

### ValueAtRisk and ConditionalValueAtRisk

```go
type VaRMethod uint8

const (
    VaRHistorical VaRMethod = iota
    VaRParametric
)

func ValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)
func ConditionalValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)
```

Both functions report a loss fraction using a positive-loss convention. `confidence = 0.95` names the loss tail below the 5th percentile, and historical VaR is the negated 5th percentile:

```text
q = Q_(1-confidence)(returns)
VaR_historical = -q
CVaR_historical = -mean({r_i | r_i <= q})
```

`VaRHistorical` uses the empirical R type-7 quantile, matching `DataList.Percentile`. `VaRParametric` assumes normal returns and uses the sample mean and sample standard deviation:

```text
z = NormPPF(1-confidence)
VaR_parametric = -(mean + z·sd)
CVaR_parametric = -(mean - sd·φ(z)/(1-confidence))
```

`confidence` must be in `(0, 1)`. VaR and CVaR are not annualized; apply a caller-selected horizon model when appropriate because square-root-of-time scaling is not generally valid for tail risk.

### SortinoRatio

```go
func SortinoRatio(returns insyra.IDataList, minimumAcceptableReturn, periodsPerYear float64) (float64, error)
```

Returns the annualized downside-adjusted performance ratio. Downside deviation is the root mean square of `min(r - MAR, 0)` over **all** periods, including periods above the minimum acceptable return:

```text
Sortino = mean(r-MAR) / sqrt(mean(min(r-MAR, 0)^2)) · √periodsPerYear
```

### CalmarRatio

```go
func CalmarRatio(equity insyra.IDataList, days int) (float64, error)
```

Returns `AnnualizedReturn(equity, days) / MaxDrawdown(equity)`. A zero maximum drawdown is rejected because the ratio is undefined.

### InformationRatio

```go
func InformationRatio(returns, benchmark insyra.IDataList, periodsPerYear float64) (float64, error)
```

Returns the annualized mean active return divided by tracking error, where `active = returns - benchmark` and tracking error is the sample standard deviation of active returns. The series must be aligned and have equal length; zero tracking error is rejected.

### DrawdownSeries

```go
func DrawdownSeries(equity insyra.IDataList) (*insyra.DataList, error)
```

Returns one non-negative drawdown fraction per equity point, `1 - equity[t]/runningPeak[t]`. A non-positive running peak produces `nil` because drawdown is undefined there. The maximum non-nil value equals `MaxDrawdown(equity)`.

## Market Exposure (CAPM)

`Beta` and `CAPM` take two already aligned `insyra.IDataList` values of per-period returns. They do not align dates, convert prices to returns, or drop cells. Use `riskFreeRate = 0` for a raw-return regression.

### CAPMResult

```go
type CAPMResult struct {
    Beta        float64 // OLS slope, market exposure
    Alpha       float64 // Jensen's alpha per period, OLS intercept on excess returns
    RSquared    float64 // coefficient of determination; NaN for a constant asset
    BetaStdErr  float64 // standard error of Beta
    AlphaStdErr float64 // standard error of Alpha
    N           int     // number of aligned observations
}
```

### Beta

```go
func Beta(asset, market insyra.IDataList) (float64, error)
```

Returns `Cov(asset, market) / Var(market)`, using the same sample (`n-1`) denominator for covariance and variance. This is the same as the one-predictor OLS slope. `asset` and `market` must be aligned per-period returns.

### CAPM

```go
func CAPM(asset, market insyra.IDataList, riskFreeRate float64) (*CAPMResult, error)
```

Regresses `asset - riskFreeRate` on `market - riskFreeRate`. `riskFreeRate` is a **per-period** rate, matching `SharpeRatio`; for daily data, convert an annual rate to a daily rate before calling. `Alpha` is also per period, so multiply it by `periodsPerYear` only when a separate annualized headline is required.

The standard errors use residual degrees of freedom `N-2`. A constant asset excess-return series is valid: `Beta` is `0`, `Alpha` is the constant, `BetaStdErr` and `AlphaStdErr` are `0`, and `RSquared` is `NaN` because total asset variance is zero.

Both functions return an error for nil input, unequal lengths, fewer than 3 observations, a zero-variance benchmark, or a non-numeric, `NaN`, or `Inf` cell. Unreadable cells are named with their series (`asset` or `market`) and one-based row number.

## Options (Black–Scholes–Merton)

`BlackScholes` prices European calls and puts with continuous dividend yield:

```go
type OptionType uint8

const (
    OptionCall OptionType = iota
    OptionPut
)

type BSInput struct {
    Spot, Strike, Rate, DividendYield, Volatility, TimeToExpiry float64
    Type OptionType
}

type BSResult struct {
    Price, Delta, Gamma, Vega, Theta, Rho float64
}

func BlackScholes(in BSInput) (*BSResult, error)
func ImpliedVolatility(price float64, in BSInput) (float64, error)
```

`Rate` and `DividendYield` are continuously compounded annual rates as
decimals, `Volatility` is annualized, and `TimeToExpiry` is in years. The
model uses:

```text
d1 = [ln(S/K) + (r - q + σ²/2)T] / (σ√T)
d2 = d1 - σ√T

Call = S·e^(-qT)·N(d1) - K·e^(-rT)·N(d2)
Put  = K·e^(-rT)·N(-d2) - S·e^(-qT)·N(-d1)
```

The greeks returned in `BSResult` are:

| Greek | Meaning and unit |
|---|---|
| `Delta` | Change in option value for one unit of spot price |
| `Gamma` | Change in delta for one unit of spot price |
| `Vega` | Derivative per unit of volatility; divide by 100 for a one-percentage-point move |
| `Theta` | `-∂V/∂T`, value lost per year; divide by 365 for a one-day figure |
| `Rho` | Change in option value for a one-unit change in the annual rate |

At `TimeToExpiry == 0`, `Price` is intrinsic value. `Delta` is its limiting
value (1 or 0 for a call, -1 or 0 for a put), and the other greeks are zero.

`ImpliedVolatility` ignores `in.Volatility`. It checks the no-arbitrage bounds,
brackets the solution on volatility `[1e-6, 10]` with bisection, then polishes
it with Newton iterations using vega. The call bounds are:

```text
max(S·e^(-qT) - K·e^(-rT), 0) ≤ price ≤ S·e^(-qT)
```

The put bounds are:

```text
max(K·e^(-rT) - S·e^(-qT), 0) ≤ price ≤ K·e^(-rT)
```

Prices outside these bounds and `TimeToExpiry == 0` return errors. The solver
also has a 200-iteration cap, so it always terminates.

## Factor Models

`FactorModel` fits a multiple-factor ordinary least-squares model to aligned per-period returns. It subtracts `riskFreeRate` from the asset return only, then regresses the result on every column of `factors`. Factor columns are taken as given: Fama–French-style `Mkt-RF`, `SMB`, and `HML` columns are already excess or long–short factors. If the market column is a raw market return, subtract the same per-period risk-free rate before passing it to `FactorModel`.

### FactorModelResult

```go
type FactorModelResult struct {
    Alpha            float64
    AlphaStdErr      float64
    AlphaTValue      float64
    AlphaPValue      float64
    FactorNames      []string
    Exposures        []float64
    StdErrs          []float64
    TValues          []float64
    PValues          []float64
    RSquared         float64
    AdjustedRSquared float64
    N                int
    Residuals        []float64
}

func (r *FactorModelResult) Exposure(name string) (float64, bool)
```

`FactorNames` follows the factor table's column order. `Exposures`, `StdErrs`, `TValues`, and `PValues` use the same indexes. `Exposure` returns the named factor's exposure and `true`, or `0` and `false` when the name is not present.

### FactorModel

```go
func FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64) (*FactorModelResult, error)
```

`FactorModel` uses the same OLS coefficients, inference values, R², adjusted R², and residuals as `stats.LinearRegression`. With one factor, pass an already excess market column and the raw market return plus `riskFreeRate` to `CAPM`; the factor-model exposure, alpha, and standard errors agree with CAPM.

The model requires at least `k + 2` observations for `k` factors. It refuses nil inputs, an empty factor table, unequal asset/factor lengths, unreadable cells, and `NaN` or `Inf` values. Errors identify the factor column and one-based row where applicable. A collinear factor set returns the regression error instead of a near-singular numeric result. Standard errors are ordinary OLS standard errors; heteroskedasticity and autocorrelation adjustments such as Newey–West are outside this API.

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

## Probabilistic Forecasting

Backtests answer "how would this rule have done"; the question that follows is "if I keep this configuration, where might I be in a year?". A single predicted return is unreliable and misleading. A **distribution** — median with a confidence band, probability of loss, drawdown spread — is defensible and states its uncertainty. `BlockBootstrap` builds that distribution by resampling the observed returns in blocks, which keeps their autocorrelation, volatility clustering, and fat tails without assuming normality, and `PercentileBands` turns the simulated paths into the bands of a fan chart.

### Types

```go
type BootstrapConfig struct {
    Horizon    int    // future periods per path (e.g. 252 trading days); > 0
    BlockSize  int    // block length in periods, 1 <= BlockSize <= len(returns);
                      // the MEAN block length when Stationary is true
    Paths      int    // number of simulated paths; > 0
    Seed       uint64 // always applies; same inputs + seed → bit-identical output
    Stationary bool   // false: moving block bootstrap; true: stationary bootstrap
}

type BootstrapResult struct {
    Returns [][]float64 // Paths × Horizon resampled per-period returns
    Equity  [][]float64 // Paths × (Horizon+1) compounded from 1.0 (Equity[p][0] == 1)
}
```

### BlockBootstrap

```go
func BlockBootstrap(returns insyra.IDataList, cfg BootstrapConfig) (*BootstrapResult, error)
```

Resamples `returns` (per-period simple returns, e.g. `0.012` for +1.2%) into `cfg.Paths` future series of `cfg.Horizon` periods and compounds each into an equity path starting at 1.0 — the same convention as `WalkForwardResult.Equity`. Use `Equity` for fan charts and `Returns` for per-path statistics (a bootstrapped Sharpe distribution, for example).

- **`Stationary: false`** (default) is the *moving block bootstrap* (Künsch 1989): every block has exactly `BlockSize` consecutive returns, its start drawn uniformly from `[0, n-BlockSize]`. Blocks are concatenated until `Horizon` values are collected, then truncated.
- **`Stationary: true`** is the *stationary bootstrap* (Politis & Romano 1994): block lengths are geometrically distributed with mean `BlockSize`, the start is drawn from the whole series, and indexing wraps past the end back to the beginning. This removes the edge effects of fixed blocks and makes the resampled series stationary.

**Seed.** `Seed` always applies and the zero value is simply seed 0 — there is no "unset means random" mode, because auditability is the point of the seed. Pass a clock-derived value yourself when you want a fresh draw. The stream is a PCG generator with the uniform reductions done inside `quant`, so a result does not depend on the Go release.

**Input.** Every element of `returns` must be a finite number. A value that cannot be read, or is NaN or Inf, is an error naming its row — it is never replaced by zero. The equity paths assume `returns >= -1` (a simple return cannot lose more than 100%).

**Choosing the parameters.** A block length around `n^(1/3)` for `n` observations is a common starting point; 20 trading days is a sensible default for daily data. Longer blocks keep more serial structure, shorter blocks give more distinct paths. A few hundred observations and a few thousand paths give stable 5%/95% bands. These are statistical recommendations, not enforced limits.

**Returns:** `(result, err)` — `err` is non-nil for an empty or unreadable series, `Horizon <= 0`, `Paths <= 0`, `BlockSize < 1`, or `BlockSize > len(returns)`. Memory is `Paths × Horizon × 16` bytes (both matrices), about 20 MB at 5000 × 252.

### PercentileBands

```go
func PercentileBands(paths [][]float64, percentiles []float64) ([][]float64, error)
```

For every time step, the requested percentiles across all paths — the vertical slices of a fan chart. `paths` is any matrix of equal-length rows (`BootstrapResult.Equity`, `BootstrapResult.Returns`, or your own). `percentiles` are on the `0..100` scale, and `bands[i]` is the series for `percentiles[i]` in the order you gave, each as long as one path.

The quantile is R's **type-7** (linear interpolation between order statistics), the same definition as `DataList.Percentile`, `Quartile`, and `Describe`, so the bands agree with the rest of the library.

**Returns:** `(bands, err)` — `err` is non-nil if `paths` is empty or ragged, or `percentiles` is empty or contains a value outside `[0, 100]`.

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

### Option pricing and implied volatility from a datafetch chain

`datafetch` returns calls and puts as `DataTable`s. Select a row from a chain,
read its `strike` and `lastPrice`, and pass the underlying spot and time to
expiry to `quant.ImpliedVolatility`:

```go
yf, err := datafetch.YFinance(datafetch.YFinanceConfig{})
if err != nil { log.Fatal(err) }
chain, err := yf.Ticker("AAPL").OptionChain("2026-12-18")
if err != nil { log.Fatal(err) }

// Choose a row from chain.Calls or chain.Puts after inspecting the table.
// These values are the selected row's strike and lastPrice columns.
strike := 200.0
lastPrice := 12.50
optionType := quant.OptionCall
iv, err := quant.ImpliedVolatility(lastPrice, quant.BSInput{
    Spot: 195.0, Strike: strike, Rate: 0.04, DividendYield: 0.00,
    TimeToExpiry: 0.5, Type: optionType,
})
if err != nil { log.Fatal(err) }
fmt.Printf("implied volatility = %.2f%%\n", iv*100)
_ = chain // use its Calls/Puts table to supply strike and lastPrice
```

### Risk report

```go
var95, _ := quant.ValueAtRisk(returns, 0.95, quant.VaRHistorical)
cvar95, _ := quant.ConditionalValueAtRisk(returns, 0.95, quant.VaRHistorical)
sortino, _ := quant.SortinoRatio(returns, 0, 252)
calmar, _ := quant.CalmarRatio(equity, 30)
drawdowns, _ := quant.DrawdownSeries(equity)

fmt.Printf("VaR95=%.2f%% CVaR95=%.2f%% Sortino=%.3f Calmar=%.3f\n",
    var95*100, cvar95*100, sortino, calmar)
_ = drawdowns // chart one value per equity period when needed
```

`ValueAtRisk` and `ConditionalValueAtRisk` are per-period tail measures. `SortinoRatio` and `CalmarRatio` annualize according to their explicit period or calendar-day arguments.

### Beta of a stock against its index

`Beta` expects returns, so align the two price tables on their date first. An inner merge avoids inventing returns for dates that appear in only one table:

```go
// Each table has a Date column and an adjusted price column. For Taiwan stocks
// that is datafetch's DailyPricesAdjusted "AdjClose"; for Yahoo Finance it is
// "Adj Close". When both tables use the same price column name, the merged table
// keeps the left table's column as "AdjClose" and renames the right one
// "AdjClose_other".
aligned, err := assetPrices.Merge(indexPrices,
    insyra.MergeDirectionHorizontal, insyra.MergeModeInner, "Date")
if err != nil {
    log.Fatal(err)
}

assetReturns := aligned.PctChangeCol("AdjClose", 1).ClearNils()       // asset
marketReturns := aligned.PctChangeCol("AdjClose_other", 1).ClearNils() // index
beta, err := quant.Beta(assetReturns, marketReturns)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("market beta = %.3f\n", beta)
```

Use adjusted prices, not raw closes. On an ex-dividend or ex-rights day the quoted price drops by the distribution without any loss to the holder, so a raw `Close` series records a fake loss on every ex-date and biases beta, CAPM alpha, VaR, and factor exposures. The date window and return frequency also change beta, so compare assets with the same dates and sampling interval. See the `CAPM` section above for alpha and standard errors.

### Three-factor attribution

The factor columns below are already aligned with the asset returns. `Mkt-RF` is the market excess return, so it is passed unchanged; `riskFreeRate` is subtracted from the asset only:

```go
factors := insyra.NewDataTable(
    aligned.GetColByName("Mkt-RF"), aligned.GetColByName("SMB"), aligned.GetColByName("HML"),
).SetColNames([]string{"MKT", "SMB", "HML"})

model, err := quant.FactorModel(assetReturns, factors, dailyRiskFreeRate)
if err != nil {
    log.Fatal(err)
}
smbExposure, _ := model.Exposure("SMB")
fmt.Printf("alpha = %.4f, SMB exposure = %.3f\n", model.Alpha, smbExposure)
```

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

### Probabilistic forecast (fan chart)

```go
// returns: an IDataList of historical daily returns, typically from a backtest.
res, err := quant.BlockBootstrap(returns, quant.BootstrapConfig{
    Horizon:   252,  // one trading year
    BlockSize: 20,   // keep ~a month of serial structure per block
    Paths:     5000,
    Seed:      42,   // reproducible, auditable
})
if err != nil {
    log.Fatal(err)
}

bands, err := quant.PercentileBands(res.Equity, []float64{5, 25, 50, 75, 95})
if err != nil {
    log.Fatal(err)
}
last := len(bands[0]) - 1
fmt.Printf("after 1y: p5=%.2f  median=%.2f  p95=%.2f (×start equity)\n",
    bands[0][last], bands[2][last], bands[4][last])

// Anything else is a few lines over the paths:
losses := 0
for _, eq := range res.Equity {
    if eq[len(eq)-1] < 1 {
        losses++
    }
}
fmt.Printf("P(loss) = %.1f%%\n", 100*float64(losses)/float64(len(res.Equity)))
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
- **`BlockBootstrap`** — empty or unreadable `returns` (non-numeric, NaN, Inf — the error names the row), non-positive `Horizon`/`Paths`, `BlockSize < 1`, `BlockSize > len(returns)`
- **`PercentileBands`** — empty or ragged `paths`, empty `percentiles`, a percentile outside `[0, 100]`
- **`Beta`** — nil input, unequal lengths, fewer than 3 returns, zero benchmark variance, or an unreadable/non-finite cell named with its series and row
- **`CAPM`** — the same input and benchmark validation as `Beta`; `riskFreeRate` is per period
- **`FactorModel`** — nil input, no factor columns, unequal asset/factor lengths, fewer than `k+2` observations, unreadable/non-finite cells named with the factor and row, or collinear factors
- **`BlackScholes`** — non-positive Spot, Strike, or Volatility; negative TimeToExpiry; non-finite inputs; or an unknown option Type
- **`ImpliedVolatility`** — non-finite price, zero TimeToExpiry, a price outside the option's named lower or upper bound, or failure to converge within 200 iterations
- **`ValueAtRisk`** — fewer than 2 returns, confidence outside `(0, 1)`, unknown method, or an unreadable/non-finite return named with its row
- **`ConditionalValueAtRisk`** — the same return, confidence, and method validation as `ValueAtRisk`
- **`SortinoRatio`** — fewer than 2 returns, non-positive periods per year, zero downside deviation, or an unreadable/non-finite return
- **`CalmarRatio`** — invalid annualized-return input, zero maximum drawdown, or an unreadable/non-finite equity point
- **`InformationRatio`** — unequal lengths, fewer than 2 observations, non-positive periods per year, zero tracking error, or an unreadable/non-finite cell
- **`DrawdownSeries`** — empty equity or an unreadable/non-finite equity point; non-positive running peaks are represented as `nil`

The package never logs warnings on its own and never panics from valid input — it follows the same error-first contract as [`stats`](./stats.md) and [`finance`](./finance.md).

---

## Related Packages

- [`stats`](./stats.md): `NormCDF` / `NormPPF` (used internally by the overfitting diagnostics), plus skewness/kurtosis, hypothesis tests, and `LinearRegression` when you need p-values or confidence intervals for market exposure
- [`DataList.Percentile`](./DataList.md): the same R type-7 quantile `PercentileBands` uses, for one-off percentiles of a single series
- [`finance`](./finance.md): high-precision TVM, NPV/IRR, bonds — use it for exact cashflow/loan math rather than return-series analytics
- [`insyra`](../README.md): `DataList` / `DataTable` core types — the input types for the performance and overfitting functions. Build a `DataList` from raw numbers with `insyra.NewDataList(vals...)`.
