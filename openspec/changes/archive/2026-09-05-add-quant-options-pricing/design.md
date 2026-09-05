# Design: add-quant-options-pricing

## Context

`stats.NormCDF(x)` is exported; the standard normal density is not. `quant` is float64, error-first, and already imports `stats`. `finance` is decimal-based and its `options.go` holds package options (`PaymentTiming`, rounding), not derivatives — the name is a coincidence and nothing there is reused. `datafetch` returns option chains with strikes, prices, and expiries that this change's functions consume.

## Goals / Non-Goals

**Goals:**
- Closed-form BSM price and five greeks, conventions stated once.
- Implied volatility that always terminates and names the bound it violated.
- Golden values from a textbook and internal consistency (parity, finite differences, round trips).

**Non-Goals:**
- American options, binomial trees, volatility surfaces, exotic payoffs, discrete dividends, day-count conversion from calendar dates to `TimeToExpiry`, batch pricing of a `DataTable` chain. Each is a separate change; the last is a natural follow-on once the scalar API exists.

## Decisions

### Lives in `quant`, not `finance`

`finance` promises decimal exactness for cashflow math. Option pricing needs `exp`, `log`, and the normal CDF, which are float64 routines; wrapping them in decimals would add cost without precision. `quant` already holds the return-analytics float64 toolkit and imports `stats`.

### Greek units: per unit σ, per year

Trading screens quote vega per 1% and theta per day; libraries (QuantLib, py_vollib) return per-unit and per-year. Returning the raw derivatives keeps the greeks consistent with the finite-difference tests and with each other; the docs give the divisors (`/100`, `/365`) for screen units.

### Implied volatility: bracket then polish

Newton alone diverges for deep out-of-the-money options where vega is tiny; bisection alone is slow. Bisection on `[1e-6, 10]` guarantees a bracket after the bound check, then Newton with vega finishes in a few steps. Iterations are capped and an error is returned if the cap is hit, so the function cannot spin.

### Expiry as a special case, not an error

`T = 0` is a legitimate query (settlement-day intrinsic). The formulas divide by `√T`, so it is handled explicitly with the limiting greeks. Implied volatility at `T = 0` is refused because every σ gives the same price.

## Risks / Trade-offs

- [Users pass percentages for rates or σ] → doc comment states decimals; the Hull example in the docs shows `0.10`, `0.20`.
- [`NormCDF` precision in the far tails] → deep OTM prices near zero; the implied-vol lower bound check uses the same function so the bracket stays consistent. Tested at `K/S = 1.5`.
- [Theta sign convention] → defined as `−∂V/∂T` (value lost per year of passing time), stated in the doc comment, verified by finite difference.

## Open Questions

None that block implementation.
