# Tasks

- [x] Tests first in `quant/momentum_portfolios_test.go` on a hand-computed four-stock panel: one-month hold, overlapping cohorts summed per stock, a stock with no return, NaN after a skipped month and around a group with no return, the spread's turnover.
- [x] `Turnover` and `LongShortTurnover` in `quant/momentum_portfolios.go`, from the weights the returns already use.
- [x] `Docs/quant.md`: the `### MomentumPortfolios` section describes both fields, the drifted-weight definition, a one-way cost example, the NaN cases, and how it differs from `BacktestSignal`'s `Turnover`; "Turnover is not reported" is gone.
- [x] Both CHANGELOGs: the unreleased `MomentumPortfolios` entry under `### `quant`` names the turnover.
- [x] `openspec validate momentum-turnover --strict` passes.
