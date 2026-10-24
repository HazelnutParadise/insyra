# Proposal: quant-ta-filter-rule

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), item 4: Alexander's filter rule is the classic trend rule that Fama and Blume (1966) used to test weak-form efficiency, and a study of it needs the rule as a position series. The owner approved the shape on 2026-10-24 after a Fable review; `quant-backtest-short` lets `BacktestSignal` take its short leg.

## What Changes

- New `ta.FilterRule(prices, threshold, opts ...FilterRuleOptions)` returning a `*insyra.DataList` of positions, 1 long, 0 flat, -1 short, one per row.
- It is built on the confirmed turning points of `ZigZag`: a confirmed low starts a long, a confirmed high a short (flat with `LongOnly`), held until the next confirmed turning point. It is flat before the first one and uses no later price.
- `FilterRuleOptions` has `LongOnly` and `SegmentStarts` (the position is flat on the first row of a segment; nothing is carried over a break).
- Errors: a threshold outside (0, 1), a non-positive or non-finite price, a nil list, more than one options value, and a segment start outside the rows.
- Not in this change: a filter on highs and lows rather than closes, transaction costs (they belong to `BacktestSignal`).

## Capabilities

### New Capabilities

- `quant-ta-filter-rule`: Alexander's filter rule as a position series.

## Impact

`quant/ta/filter_rule.go`, `quant/ta/filter_rule_test.go`, `quant/signal_backtest_test.go`, `Docs/quant-ta.md`, both CHANGELOGs. Additive.
