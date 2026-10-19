# Proposal: quant-forward-returns

## Why

[#445](https://github.com/HazelnutParadise/insyra/issues/445) and [#446](https://github.com/HazelnutParadise/insyra/issues/446): testing a signal on a stock panel means measuring the excess return after each event, or after each bucket of a signal, with a t-test that does not count many same-day observations as independent. Both studies were written in pandas, and they share the same forward-return core.

## What Changes

- `quant.EventStudy(prices, events, dates, opts)` and `quant.SignalBuckets(signal, prices, dates, opts)` share one panel core: entry `Delay` rows after the signal (at least 1), several horizons, three benchmarks (cross-section mean, index levels, a series' own mean), a tradable mask, monthly averaging and a Newey-West t-test on the monthly means.
- `EventStudy` adds same-stock de-duplication (`MinGap`); `SignalBuckets` takes edges or pooled quantiles.
- `quant.Bias` and `quant.BiasZ` (the deviation from the moving average, and that deviation over its own earlier volatility, never using the current day).
- Checked by hand-computed panels, a random-walk panel where no bucket is significant, and a test that `BiasZ` ignores later prices. The comparison with the pandas reference script in the issue was not made, since that script is not in this repository.

## Capabilities

### New Capabilities

- `quant-forward-returns`: forward excess returns by event or by signal bucket.

## Impact

`quant/forward_returns.go`, `quant/event_study.go`, tests, `Docs/quant.md`, both CHANGELOGs.
