# Proposal: throttle-spacing-contract

## Why

`TestTWStockThrottle` failed on CI twice on 2026-09-25 and 2026-09-26, once on windows-latest and once on macOS-latest (PR #391, run 36181766425). Both times the fixture transport saw the two requests 19.98 ms apart against a 20 ms `Interval`.

The limiter behaved as designed. `IntervalLimiter.Wait` schedules each call's slot from `time.Now()` read at the call, spaces successive slots by the interval, and never lets a call return before its slot. So it guarantees spacing between scheduled starts. The transport stamps each request later, after the request is built and handed to `net/http`, and that overhead differs between two requests by a few microseconds. A gap measured there can therefore fall microseconds short of the interval.

No client-side limiter can promise spacing at a point it does not control. The send instant lies inside `net/http`, and the server sees the network's jitter on top of that. The spec scenario, the `Interval` doc comments and `Docs/datafetch.md` still promise spacing between requests as sent, which nothing enforces. One limiter test absorbs the same gap with an unexplained half-interval of slack.

## What Changes

- The contract becomes what the limiter guarantees. Successive requests' scheduled starts are at least `Interval` apart, and no request starts before its scheduled time. For the k-th request to start, counted from 1, it follows that it starts no earlier than (k−1)·`Interval` after a moment taken before the first one was scheduled. A run of n requests therefore takes at least (n−1)·`Interval`.
- `TestTWStockThrottle` takes a timestamp before the first request and asserts that the second request reaches the transport at least `Interval` after it. This bound holds exactly, with no tolerance. It fails if throttling is removed.
- `TestWait_AllowedCallsAreNeverCloserThanTheInterval` replaces its half-interval slack with the same exact bound on the sorted release times.
- The limiter's type comment, the `Interval` fields of `TWStockConfig`, `YFinanceConfig` and `TWGeocodingConfig`, and the three `Interval` rows of `Docs/datafetch.md` say the same thing.
- The limiter's behaviour does not change, so there is no CHANGELOG entry.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `datafetch-twstock`: the throttle scenario states the spacing the client can guarantee.

## Impact

- `datafetch/twstock_test.go`, `datafetch/internal/limiter/interval_limiter_test.go`, the comments in `datafetch/internal/limiter/interval_limiter.go`, `datafetch/twstock.go`, `datafetch/yfinance.go` and `datafetch/geocoding.go`, and `Docs/datafetch.md`.
