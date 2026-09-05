# Proposal: add-quant-options-pricing

## Why

`datafetch` already returns option chains and `finance` prices bonds, but nothing in Insyra prices an option or reports its sensitivities. Black–Scholes–Merton with greeks and implied volatility is the closed-form baseline every derivatives desk and every retail option screen starts from. It is pure arithmetic over `stats.NormCDF`, so it belongs in `quant` (float64), not in the decimal-precision `finance` package.

## What Changes

- New `quant.OptionType` (`OptionCall`, `OptionPut`) and `quant.BSInput{Spot, Strike, Rate, DividendYield, Volatility, TimeToExpiry float64; Type OptionType}`. `Rate` and `DividendYield` are continuously compounded annual rates; `TimeToExpiry` is in years; `Volatility` is annualized.
- New `quant.BlackScholes(in BSInput) (*BSResult, error)` returning `Price`, `Delta`, `Gamma`, `Vega`, `Theta`, `Rho`. Vega is per unit of volatility (not per 1%), theta per year (not per day); the docs state both and give the divisors. At `TimeToExpiry == 0` the price is intrinsic value and greeks are their limits (delta 0 or ±1, others 0).
- New `quant.ImpliedVolatility(price float64, in BSInput) (float64, error)`: solves for the volatility that reproduces `price`, bracketing with bisection on `[1e-6, 10]` then polishing with Newton on vega, to `1e-10` in price. A price outside the no-arbitrage bounds (below intrinsic, above the spot-or-strike ceiling) is an error naming the bound.
- Validation: `Spot`, `Strike`, `Volatility > 0`, `TimeToExpiry >= 0`, finite inputs, a known `Type`.
- Golden tests use Hull's textbook example (S=42, K=40, r=0.10, σ=0.20, T=0.5: call 4.76, put 0.81) and put–call parity, delta/gamma finite-difference checks, and implied-volatility round trips.
- Docs (`Docs/quant.md`, README rows), `skills/insyra`, and both changelogs are updated in the same change.

## Capabilities

### New Capabilities

- `quant-options`: Black–Scholes–Merton option price, greeks, and implied volatility.

### Modified Capabilities

(none)

## Impact

- New files `quant/options.go`, `quant/options_test.go`. Uses `stats.NormCDF` (already a `quant` dependency) and a local standard-normal density.
- `Docs/quant.md` (overview bullet, "Options" section, usage example, error rows), `Docs/README.md`, `README.md`, `README_TW.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `quant/init.go`.
- No new dependencies, no breaking changes.
