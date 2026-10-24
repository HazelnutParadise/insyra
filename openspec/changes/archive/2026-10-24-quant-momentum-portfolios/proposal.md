# Proposal: quant-momentum-portfolios

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), item 5: a momentum study needs Jegadeesh and Titman's (1993) overlapping portfolios, and `QuantileSort` sorts one date at a time and cannot carry a formation period or overlapping holds. The owner approved the shape on 2026-10-24 after a Fable review.

## What Changes

- New `quant.MomentumPortfolios(table, dateCol, stockCol, returnCol, formation, groups, opts ...MomentumOptions)` on a long-form panel like `QuantileSort`, returning a `MomentumResult` with `Groups`, `LongShort` and `Skipped`.
- Each month ranks stocks by the compounded return over `formation` months, ending `Skip` months back; cuts them into `groups` groups with `QuantileSort`'s rank-tie rule; holds each group for `Hold` months and averages the overlapping portfolios with equal weights.
- `MomentumOptions` has `Skip` (default 0) and `Hold` (default 1).
- A month is reported once `Hold` portfolios have been formed; a month whose portfolio could not be formed is listed in `Skipped`.
- Errors: `formation` below 1, `groups` below 2, a negative `Skip` or `Hold`, more than one options value, a missing column or value, a stock twice in a month, and too few months.
- Not in this change: turnover (it needs weights carried forward with returns), weighting other than equal, a rolling formation on non-consecutive dates.

## Capabilities

### New Capabilities

- `quant-momentum-portfolios`: overlapping momentum portfolios.

## Impact

`quant/momentum_portfolios.go`, `quant/momentum_portfolios_test.go`, `Docs/quant.md`, both CHANGELOGs. Additive.
