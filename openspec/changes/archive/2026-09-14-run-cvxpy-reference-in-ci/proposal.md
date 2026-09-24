## Why

`quant/portfolio_cvxpy_test.go` checks `OptimizePortfolioMoments` against cvxpy, an independent convex solver. It is opt-in because cvxpy is usually absent, and strict mode is meant to turn it on. But the Reference Verification workflow, the one job that runs with strict mode, never runs `./quant/` and never installs cvxpy, so the comparison has not run in CI once (#302, TS-4). A skip and a pass look the same, which is exactly what that workflow exists to prevent.

Measured locally on 2026-09-14 with cvxpy 1.7.5: the test passes in about four seconds, with a largest objective difference of 4.09e-09 over 40 problems.

## What Changes

- Reference Verification installs `cvxpy` with the other Python reference implementations, and its interpreter check imports it.
- A new step runs `TestPortfolioAgreesWithCVXPY`. The job already sets `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1`, so if cvxpy stops installing, the job fails rather than skipping.
- `ENG.md` lists cvxpy among the Python references the strict gate expects.

No library code or user-visible behaviour changes, so there is no changelog entry.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `verification-integrity`: the requirement that CI provides the toolchains its gates need gains a scenario for an opt-in comparison that no workflow step runs.

## Impact

- `.github/workflows/reference-verification.yml`, `ENG.md`.
- `api-review.md` TS-4, `delivery-status.md`, issue #302.

## Backport to dev (0.3.x)
Backported in full. The workflow, `ENG.md` and the spec on this line are identical to `0.4`'s before this change, so nothing needed adapting. `api-review.md` is a `0.4` tracker and is not on this line, so `0052ae06`'s edit to it was dropped.
