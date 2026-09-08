# Design: add-quant-beta-capm

## Context

`quant` takes raw series as `insyra.IDataList`, returns errors rather than logging, validates rather than defaults, and pairs each exported function with an unexported `...F64` core for tests. It imports only the root package and `gonum/stat`. `add-quant-block-bootstrap` added `numericSeries(dl, label)`, which reads under `AtomicDo`, converts with `insyra.ToFloat64Safe`, and refuses non-finite values with the row named; ENG.md records that `DataList.ToF64Slice` fabricates zeros and new numeric analyses must not use it. `stats.LinearRegression` is R-verified and already yields the slope, intercept, standard errors, and R² of a one-predictor OLS, but its result is general-purpose (`Coefficients`, `Residuals`, confidence intervals) and `stats` refuses the leading `nil` that `DataList.PctChange` emits.

Measured on 2026-09-03 with a synthetic price pair: `History → Merge on Date → PctChangeCol → ClearNils → stats.LinearRegression(rs, rm).Slope` gives 1.5641, identical to `stats.Covariance / Var`. The math exists; the ticket is the named entry point.

## Goals / Non-Goals

**Goals:**
- One call answers "what is this stock's beta" from two aligned return series.
- A second call gives the full single-index fit: alpha, R², standard errors, N.
- Results agree exactly with `stats.LinearRegression` and are pinned by a hand-computed golden.
- Invalid or unreadable input is refused, never coerced.

**Non-Goals:**
- Date alignment or price-to-return conversion inside `quant`. `Merge` and `PctChangeCol` already do this, and a from-prices entry point would have to choose `Close` versus `Adj Close` and a fill policy silently. The docs carry the recipe and the choices instead.
- Rolling beta, exponentially weighted beta, multi-factor models (Fama–French), p-values or confidence intervals on beta. Each is a separate change; users who need inference call `stats.LinearRegression` directly.
- Annualizing alpha. Alpha is per period; the docs say to multiply by `periodsPerYear` for a headline figure, the same way `SharpeRatio` documents its annualization.
- Moving existing `quant` functions off `ToF64Slice` (standing follow-up in AGENTS.md).

## Decisions

### Two functions: `Beta` and `CAPM`

`Beta(asset, market)` returns a `float64`; `CAPM(asset, market, rf)` returns `*CAPMResult`. A single function returning a struct would make the common question (`beta, err := quant.Beta(rs, rm)`) read a field off a result, and a single scalar function would drop alpha and the standard errors that make the number defensible. Two functions match how `SharpeRatio` (scalar) and `WalkForward` (result struct) already coexist. `Beta` does not take `rf`: subtracting one constant from both series changes neither covariance nor variance, so `Beta == CAPM(...).Beta` for every `rf`, and a test asserts it.

### Compute the OLS directly, do not call `stats`

`quant` stays free of a `stats` import. One-predictor OLS is closed-form: `β = Sxy / Sxx`, `α = ȳ − β·x̄`, `s² = SSR / (n−2)`, `se(β) = √(s² / Sxx)`, `se(α) = √(s² · (1/n + x̄² / Sxx))`, `R² = 1 − SSR / SST`. Twenty lines, no matrix, no gonum `mat`. The test file imports `stats` and asserts agreement with `LinearRegression` on random data to 1e-12, so the two cannot drift silently, and a golden test with hand-computed values pins the arithmetic independently of both. Alternative considered: wrapping `stats.LinearRegression` and copying fields — simpler code but a package dependency for a formula, and `stats` computes residuals, t-values, and intervals this ticket does not expose.

### Excess returns, per-period `rf`

`CAPM` regresses `asset − rf` on `market − rf`. Regressing raw returns and subtracting `rf` afterwards gives a different alpha (`α_raw − rf·(1−β)`), and the excess-return form is the textbook CAPM definition. `rf` is per period, matching `SharpeRatio`; the doc comment says so and gives the daily conversion.

### Degenerate cases: refuse the benchmark, allow the asset

A benchmark with zero variance makes beta `0/0`; that is refused. An asset with zero variance is a legitimate cash-like series: `β = 0`, `α` is the constant, and `R²` is `0/0`. Following `stats`' precedent of `NaN` for undefined inference rather than a zero fallback, `RSquared` is `NaN` and both standard errors are 0 (residuals are exactly zero). This is documented and tested rather than turned into an error, so a portfolio that includes cash does not fail the loop.

### Input through `numericSeries`, lengths checked before conversion errors

Both series are converted with the existing helper, labelled `asset` and `market` so the error names the offending series and row. Length mismatch is checked first, the way `stats.gatherRegressionInputs` does, so a caller who passed differently sized inputs hears about that rather than about a cell. Minimum `n` is 3: fewer leaves no residual degree of freedom for the standard errors, and a two-point beta is a line through two points.

## Risks / Trade-offs

- [Users pass prices instead of returns] → the doc comment and `Docs/quant.md` state the input is per-period returns; a beta of prices is a recognizable nonsense number, but the API cannot detect it. The docs example shows the conversion.
- [Users pass misaligned series of equal length] → undetectable inside `quant`; the docs recipe uses an inner `Merge` on the date column before `PctChangeCol` so the lengths only match when the dates do.
- [Duplicated OLS formula drifts from `stats`] → the 1e-12 agreement test fails the build the moment they differ.
- [`NaN` R² for a constant asset surprises a caller summing R² across assets] → documented in the result's doc comment and the error-handling section; `math.IsNaN` is the check.

## Open Questions

None that block implementation.
