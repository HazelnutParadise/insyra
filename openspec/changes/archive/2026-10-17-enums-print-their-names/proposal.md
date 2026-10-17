# Proposal: enums-print-their-names

## Why

insyra has 65 exported enum types written three ways: 40 `string`, 19 `int`, 6 `uint8` ([#246](https://github.com/HazelnutParadise/insyra/issues/246), api-review QU-1). Measured on `0.4` at 8f3399dd, 19 of the 25 numeric enums have no `String()`, so they print as bare numbers. `fmt.Print(quant.VaRParametric, quant.OptionPut, quant.MaximumSharpe)` prints `1 1 2`. `insyra.MergeModeOuter` and `insyra.UnknownError` both print `1`. Errors use `%d`, so `ValueAtRisk(r, 0.95, VaRMethod(3))` says `VaR: unknown method 3`. Two existing `String()` methods also misreport unknown values: `ml.ClassAverage(9)` prints `macro`, and `ml.MetricDirection(9)` prints `no direction`.

The owner ruled on 2026-10-18 to keep every underlying type and to make every enum readable. No enum is serialized anywhere, so moving to one underlying type would break callers on one side and gain no readable storage.

## What Changes

- Every `int` and `uint8` enum has `String()`. A known value prints its constant's name, or the word an existing `String()` already printed. An unknown value prints `TypeName(n)`.
- Every error that reports an enum value prints it with `%v`.
- **BREAKING**: `quant.VaRMethod`, `quant.OptionType`, `quant.PortfolioObjective`, `finance.DayCountBasis`, `finance.PaymentTiming` and `datafetch.GoogleMapsStoreReviewSortBy` change from `uint8` to `int`. Constants, untyped literals and conversions still compile. Only arithmetic that mixes one of them with a `uint8` value breaks.
- Printing a numeric enum with `%v` now gives its name instead of a number.
- AGENTS.md records the rule for choosing an enum's underlying type. The two types that break the rule are recorded as a follow-up.

## Capabilities

### New Capabilities

- `enum-conventions`: how an enum's underlying type is chosen and how an enum value prints.

## Impact

Root package, `quant`, `finance`, `datafetch`, `parquet`, `stats`, `ml`. Updates `Docs/` pages that show these types, both CHANGELOGs, `AGENTS.md`, `api-review.md` and `delivery-status.md`.
