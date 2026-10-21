# Proposal: quant-abnormal-returns

## Why

[#478 策略檢定補齊](https://github.com/HazelnutParadise/insyra/issues/478), part 4: `EventStudy` measures returns only after the entry, so it cannot show whether news leaked before an event or how a price drifted around it. The usual event study estimates how the stock normally moves with the market from a window before the event, then reports the gap from that normal (abnormal return) and its running total (CAR) for each day around the event.

## What Changes

- `quant.AbnormalReturns(prices, events, dates, opts)` reports, for each relative day of an event window, the mean abnormal return and mean CAR across events, a cross-sectional t-test with a two-tailed p for each, and a `CARs` table with one row per used event.
- Two expected-return models, `AbnormalMarketAdjusted` and `AbnormalMarketModel` (OLS alpha and beta on an estimation window). `MinEstimation` has no default and must be at least 3 for the market model.
- An event with a missing price or market level on any row of its window is dropped whole, so every day uses the same events. Estimation rows inside another event's window of the same stock are left out.
- The price-table reading in `newPanel` moves into a helper that both studies share; `EventStudy` and `SignalBuckets` do not change.
- Patell, BMP and Kolari-Pynnonen tests are not implemented.

## Capabilities

### New Capabilities

- `quant-abnormal-returns`: abnormal returns and CAR around events.

## Impact

`quant/abnormal_returns.go`, `quant/forward_returns.go` (helper only), `quant/enum_names.go`, tests and a numpy/scipy baseline under `quant/testdata`, `Docs/quant.md`, both CHANGELOGs.
