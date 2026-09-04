# Design: add-quant-risk-metrics

## Context

`quant/performance.go` has `sharpeRatioF64`, `maxDrawdownF64`, `annualizedReturnF64` (exported wrappers still call `ToF64Slice`, a standing follow-up). `quant/bootstrap.go` has `numericSeries` (refusing unreadable input) and `quantileType7` over a sorted slice, tested to agree with `DataList.Percentile`. `quant/overfitting.go` already imports `stats` for `NormCDF`/`NormPPF`. Package conventions: error-first, validate not default, `...F64` cores for tests, per-period inputs with an explicit `periodsPerYear`.

## Goals / Non-Goals

**Goals:**
- Tail-risk numbers (VaR, CVaR) with the two methods a risk desk expects, sign convention stated once.
- Downside (Sortino), drawdown-relative (Calmar), and benchmark-relative (information ratio) ratios beside Sharpe.
- The drawdown series users chart, so `MaxDrawdown` stops being the only view of it.

**Non-Goals:**
- Cornish–Fisher, Student-t, or Monte Carlo VaR; portfolio-level VaR from a covariance matrix; backtesting VaR exceptions (Kupiec). Each is its own change.
- Annualizing VaR by √t scaling; the docs explain why and leave it to the caller.
- Changing existing functions off `ToF64Slice`.

## Decisions

### VaR is a positive loss at a confidence level

Sign conventions differ across libraries; the package picks one and states it in every doc comment: `confidence = 0.95` names the loss exceeded 5% of the time, returned as a positive fraction. Historical VaR reuses `quantileType7` so it agrees with `DataList.Percentile` and `PercentileBands`. Parametric CVaR uses the closed form `mean − sd·φ(z)/(1−c)` with a local `φ` (standard normal density), because `stats` exports the CDF and quantile but not the density.

### Sortino divides by downside deviation over all periods

Two definitions circulate; averaging squared shortfalls over all periods (Sortino & van der Meer 1991) is the one in the CFA curriculum and in most backtesting tools, and the one that reduces sensibly as losing periods become rarer. The spec pins it with a hand-computed case so the choice cannot silently flip.

### Calmar and drawdown series compose the existing cores

`CalmarRatio` calls `annualizedReturnF64` and `maxDrawdownF64` on one converted series, so its numbers equal what a user would get by calling the two public functions. `DrawdownSeries` is the loop inside `maxDrawdownF64` made visible; a test asserts `max(series) == MaxDrawdown`.

### Input through `numericSeries` for the new functions only

New functions refuse unreadable input, as `BlockBootstrap` and `CAPM` do. The old wrappers keep `ToF64Slice` for now because changing their error behaviour is the follow-up recorded in AGENTS.md, not this ticket.

## Risks / Trade-offs

- [Users mix up the sign or the confidence direction] → doc comments give a worked example (`0.95 → 5th percentile → positive number`).
- [Parametric VaR under-states fat tails] → documented next to the method enum; historical is listed first.
- [Sortino definition disagrees with a user's spreadsheet] → the doc names the definition and the alternative, and the spec scenario pins the formula.

## Open Questions

None that block implementation.
