# Design: add-quant-portfolio-optimization

## Context

`quant` is float64, error-first, validates rather than defaults, and already imports `stats` and `gonum/stat`; `stats` imports `gonum/mat`. `lp` solves linear programs through `lpgen` and cannot express `wᵀΣw`. The 2026-09-05 decision was a pure-Go solver over a Python bridge: no new dependency, works in every Insyra build, at the cost of supporting only the constraint shapes a small solver handles well. The crosslang venv at `~/.cache/insyra-crosslang-venv` hosts Python for gated parity tests; `ml` gates `sklearn` comparisons behind `INSYRA_RUN_ML_SKLEARN`.

## Goals / Non-Goals

**Goals:**
- Minimum-variance, target-return, and maximum-Sharpe portfolios with sum-to-one and per-asset bounds.
- A frontier sweep and a moments-based entry so users can plug in their own μ and Σ.
- Correctness pinned against closed forms, brute force, and (opt-in) `cvxpy`.

**Non-Goals:**
- General linear constraints (sector caps, turnover), cardinality, transaction costs, robust/Black–Litterman estimation, shrinkage estimators. Each is a later change; the moments entry point is the seam for estimator work.
- A general QP solver API. The solver is internal to `quant`.

## Decisions

### Projected gradient with exact bounded-simplex projection

The feasible set `{w : Σw = 1, lo ≤ w ≤ hi}` admits an exact Euclidean projection: clip `w − τ·1` to the bounds and bisect on τ until the clipped vector sums to 1 (monotone in τ, converges in ~60 iterations to machine precision). With that projection, accelerated projected gradient on the convex quadratic converges linearly for positive-definite Σ and sublinearly otherwise, needs no matrix factorisation, and is ~150 lines. Alternatives considered: an active-set QP (exact in finite steps but 3–4× the code and fiddly degeneracy handling) and interior-point (overkill at these sizes). Step size `1/L` with `L` the largest eigenvalue of Σ from power iteration; restart momentum when the objective rises.

### Target return via augmented Lagrangian on top of the same projection

Adding `μᵀw = r` breaks the closed-form projection. The solver keeps the box-simplex projection and enforces the return equality with an augmented Lagrangian term `ρ/2 (μᵀw − r)² + ν (μᵀw − r)`, updating ν after each inner solve and increasing ρ until the violation is below tolerance. Attainability is checked first by solving two tiny LPs analytically (min and max of `μᵀw` over the box-simplex are greedy fills), so an unattainable target is an error before any iteration.

### Maximum Sharpe by golden-section over the frontier

Under long-only and box bounds the usual `y = w/κ` transformation does not preserve the box; sweeping the frontier does. Sharpe along the efficient frontier is unimodal in the target return, so golden-section search on `[r_minvar, r_max]` with the target-return solver at each probe converges in ~40 solves. The spec pins it against a 50-point sweep.

### Moments-based entry point and PSD check

`OptimizePortfolioMoments` lets users pass shrunk or forecast moments. Its covariance is checked symmetric and PSD (Cholesky with a small tolerance, or the minimum eigenvalue from power iteration on `λ_max·I − Σ`) so a bad matrix is refused rather than sending the solver into a non-convex problem.

### Cross-language test as opt-in with fixtures

The gated test writes random problems to JSON, runs a small Python script with `cvxpy` in the crosslang venv, and compares objective values; it records nothing into the repo, so CI without Python is unaffected. If the implementer finds the live call too slow, the fallback is a generator script that writes a fixture the ungated test reads, in the `gen_window_fixtures.py` style — either satisfies the spec as long as the ungated suite needs no Python.

## Risks / Trade-offs

- [Near-singular Σ (highly correlated assets)] → convergence slows; `Converged: false` with the current best is reported, and the docs recommend the moments entry with a shrunk covariance.
- [Frontier sweep cost] → `points` × one solve; at desk sizes (≤ 50 assets, ≤ 100 points) this is milliseconds.
- [Users read `SharpeRatio` as annualized] → the field is per period like the input; docs give the `√periodsPerYear` conversion, matching `SharpeRatio`'s documentation.
- [`cvxpy` not installed] → test skips with a clear message; installing it into the crosslang venv is documented in the test file header.

## Open Questions

None that block implementation.
