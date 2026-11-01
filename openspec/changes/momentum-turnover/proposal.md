# Proposal: momentum-turnover

## Why

[#499](https://github.com/HazelnutParadise/insyra/issues/499), momentum portfolios: a momentum return is only meaningful net of trading costs, and Jegadeesh and Titman's overlapping portfolios trade a share of each group every month that a caller cannot reconstruct from `Groups`. `Docs/quant.md` says turnover is not reported. The owner approved the design on 2026-11-01 after a Fable review.

## What Changes

- `MomentumResult` gains `Turnover *insyra.DataTable`, with the rows and columns of `Groups` (`date`, `group_1` to `group_n`), and `LongShortTurnover *insyra.DataList`, `group_n` plus `group_1` of `Turnover`.
- A group's turnover in month m is `sum_i |w_(i,m) - w_(i,m-1)+|` (DeMiguel, Garlappi and Uppal 2009, RFS 22(5)). `w_(i,m)` is the weight the return code already uses: over the Hold overlapping cohorts, `1/Hold` divided by the number of that cohort's group stocks with a return in m, summed per stock. `w_(i,m-1)+` is the weight of month m-1 moved by `(1 + r_(i,m-1))` and renormalised.
- NaN in the first reported month, in a month after a skipped one, when the group return is NaN in m or m-1, and when the drifted weights sum to zero or less.
- A stock with no return in a month is sold at its start and bought back when it returns; one that stops trading is sold in full.
- `Docs/quant.md` explains the definition, its use with a one-way cost (Novy-Marx and Velikov 2016), and how it differs from `BacktestSignal`'s `Turnover`, and drops "Turnover is not reported".

## Capabilities

### Modified Capabilities

- `quant-momentum-portfolios`: reports each group's turnover and the spread's.

## Impact

`quant/momentum_portfolios.go`, `quant/momentum_portfolios_test.go`, `Docs/quant.md`, both CHANGELOGs. Additive: `MomentumPortfolios` is unreleased and every existing field keeps its value.
