# Proposal: enums-print-their-names

## Why

insyra has 65 exported enum types, and 19 of the 25 numeric ones have no `String()`, so they print as bare numbers. `fmt.Print(quant.VaRParametric, quant.OptionPut, quant.MaximumSharpe)` prints `1 1 2`, and `insyra.MergeModeOuter` and `insyra.UnknownError` both print `1`. Errors use `%d`, so `ValueAtRisk(r, 0.95, VaRMethod(3))` says `VaR: unknown method 3`. Two existing `String()` methods also misreport unknown values: `ml.ClassAverage(9)` prints `macro`, and `ml.MetricDirection(9)` prints `no direction`.

This is the part of `0.4`'s `enums-print-their-names` that changes no type. The `0.4` change also moves six `uint8` enums to `int`, which breaks arithmetic that mixes them with a `uint8`, so that half stays on `0.4`.

## What Changes

- Every `int` and `uint8` enum has `String()`. A known value prints its constant's name, or the word an existing `String()` already printed. An unknown value prints `TypeName(n)`.
- Every error that reports an enum value prints it with `%v`.
- Printing a numeric enum with `%v` gives its name instead of a number.
- AGENTS.md records the rule.
- No underlying type changes.

## Capabilities

### New Capabilities

- `enum-conventions`: how an enum value prints.

## Impact

Root package, `quant`, `finance`, `datafetch`, `parquet`, `stats`, `ml`. Updates `Docs/Configuration.md`, both CHANGELOGs and `AGENTS.md`.
