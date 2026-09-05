# Proposal: add-quant-portfolio-optimization

## Why

Every other desk question now has an answer in `quant` — how risky, how exposed, what an option is worth — except the one that turns those numbers into a decision: how much of each asset to hold. Mean-variance optimisation is the baseline for that, and Insyra has no quadratic solver: `lp` is linear only. The decision on 2026-09-05 was a small pure-Go solver rather than a Python bridge, so the package stays dependency-free and works wherever Insyra builds; the price is a deliberately narrow constraint set.

## What Changes

- New `quant.PortfolioConfig{Objective PortfolioObjective; TargetReturn float64; RiskFreeRate float64; MinWeight, MaxWeight []float64; Tolerance float64; MaxIterations int}` and `quant.PortfolioObjective` (`MinimumVariance`, `TargetReturn`, `MaximumSharpe`). Weights are constrained to sum to 1 and to lie in `[MinWeight[i], MaxWeight[i]]`; defaults are long-only `[0, 1]`. Short positions are allowed only by passing a negative `MinWeight`.
- New `quant.OptimizePortfolio(returns insyra.IDataTable, cfg PortfolioConfig) (*PortfolioResult, error)`: columns are assets, rows are aligned per-period returns. Expected returns are column means and the covariance is the sample covariance (n−1). `PortfolioResult` carries `Weights []float64` (column order), `AssetNames []string`, `ExpectedReturn`, `Variance`, `Volatility`, `SharpeRatio` (per period, `(ExpectedReturn − rf) / Volatility`), `Iterations int`, `Converged bool`, and a `Weight(name string) (float64, bool)` lookup.
- New `quant.OptimizePortfolioMoments(mean []float64, cov [][]float64, names []string, cfg PortfolioConfig)`: the same solver on caller-supplied moments, for users who shrink or forecast their own.
- New `quant.EfficientFrontier(returns insyra.IDataTable, points int, cfg PortfolioConfig) ([]PortfolioResult, error)`: `points` target-return optimisations evenly spaced between the minimum-variance return and the maximum attainable return under the bounds.
- Solver: projected gradient descent on `½ wᵀΣw − λ μᵀw` with exact Euclidean projection onto the bounded simplex `{w : Σw = 1, lo ≤ w ≤ hi}` (bisection on the simplex multiplier), Nesterov momentum, step size from the largest eigenvalue of Σ via power iteration, stopping when the projected-gradient norm falls below `Tolerance` (default `1e-10`) or `MaxIterations` (default `10000`) is hit. `TargetReturn` is handled as an equality via the same projection extended with the return constraint through an augmented Lagrangian; `MaximumSharpe` is found by golden-section search over target returns along the frontier, using the fact that Sharpe is unimodal along the efficient frontier.
- Validation, not defaults: at least 2 assets and `n ≥ assets + 1` observations; bounds of the right length with `lo ≤ hi`, `Σlo ≤ 1 ≤ Σhi`; a target return inside the attainable range; unreadable cells refused via `numericSeries` with the column named; a non-positive-semidefinite covariance (from caller-supplied moments) refused. Non-convergence is reported through `Converged: false`, never hidden.
- Correctness is pinned three ways: (1) the unconstrained-interior case equals the closed form `Σ⁻¹1 / 1ᵀΣ⁻¹1` (and the target-return closed form) to 1e-8; (2) a 3-asset long-only case agrees with an exhaustive grid search to grid resolution; (3) an env-gated cross-language test (`INSYRA_RUN_CVXPY=1`, same pattern as the `sklearn`-gated `ml` tests) agrees with `cvxpy` on random long-only and box-bounded problems to 1e-6 in objective value.
- Docs (`Docs/quant.md`, README rows), `skills/insyra`, and both changelogs are updated in the same change. CLI exposure is a separate change.

## Capabilities

### New Capabilities

- `quant-portfolio`: mean-variance portfolio optimisation with sum-to-one and box constraints, three objectives, an efficient-frontier sweep, and a moments-based entry point, solved by a pure-Go projected-gradient method.

### Modified Capabilities

(none)

## Impact

- New files `quant/portfolio.go`, `quant/portfolio_solver.go`, `quant/portfolio_test.go`, `quant/portfolio_cvxpy_test.go` (gated), and `quant/testdata/gen_portfolio_fixtures.py` if the gated test uses recorded fixtures instead of a live Python call.
- `Docs/quant.md` (overview bullet, "Portfolio Optimization" section, usage example, error rows), `Docs/README.md`, `README.md`, `README_TW.md`, `skills/insyra/SKILL.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `quant/init.go`.
- Uses `gonum/mat` (already a dependency of `stats`) for the covariance and the closed-form checks; no new dependencies. `cvxpy` is a test-only, opt-in requirement installed into the existing crosslang venv.
