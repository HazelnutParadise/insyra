# Proposal: factor-parity-compares-invariants

## Why

The strict R parity suite compares Phi, structure and factor scores against psych element by element at 2e-5 whenever the aligned loadings already agree within 2e-5, and at 5e-3 only when the loadings differ by more. That is backwards: on a flat rotation criterion the closer the loadings, the stricter Phi is held, although Phi moves more than the loadings along a minimum. At seeded baselines, 308 of the suite's 691 failing leaves are these fields, in 125 combinations.

Measured on 2026-09-17:

- The failures are the reference's own spread, not the port. In all 125 combinations, L·Phi·L' and S·L', which a rotation leaves unchanged, agree with psych to 6.2e-6. GPArotation 2026.8.2 from 40 random starts that all reach the same minimum spreads by 2.1e-4 in the loadings and 3.1e-3 in Phi on `two_blocks` with ML and oblimin, and by 4.7e-3 and 6.0e-3 on `three_blocks` with MINRES and bentlerQ; psych picks among such starts by hyperplane count. Even the PCA/geomin case spreads by 9.0e-6 in the loadings and 9.1e-5 in Phi, so the current rule would fail two runs of R against each other.
- Research into R, Python and commercial factor analysis software before this change found no package that loosens Phi alone: GPArotation and EFAtools compare loadings and Phi at one tolerance and loosen both together for the same minimum reached by different paths (1e-3) or a flat criterion (bentlerQ); none compares structure or scores across starts. The literature decides "the same solution" by the criterion and the loadings, and a criterion value alone cannot: two solutions in Nguyen & Waller (2023) share one to six digits and still differ.
- 5e-3 is not enough for the reference's spread: bentlerQ on `three_blocks` reaches 6.0e-3 in Phi against psych while its L·Phi·L' agrees to 1e-15.
- The current rule decides "same minimum" by the criterion first and skips every field when our criterion is lower, so a different lower minimum is not checked at all, not even for being the same fitted model.

## What Changes

- The loadings decide whether our solution and psych's are the same solution: aligned loadings within `factorRotationTol`. Then every factor-frame field is compared at `factorRotationTol`, whether or not the loadings are already within 2e-5, and our criterion must be no worse than psych's. Loadings further apart pass only as a different minimum with a lower criterion, and fail otherwise.
- `factorRotationTol` goes from 5e-3 to 1e-2, measured against GPArotation's same-minimum spread (6.0e-3 at most) and below the 0.02 that EFAtools and Grieder & Steiner treat as a practically small difference.
- New strict fields for every rotation and verdict, a different minimum and Promax included: L·Phi·L' and S·L' against psych at 2e-5, W·L' for regression and Bartlett scores at `scoreInvariantTol` = 1e-4 (4.5e-5 measured, the extraction's 6e-6 through R⁻¹), and our structure equal to loadings × Phi to 1e-10.
- A unit test without R shows the invariants stay put under rotation, factor reordering and sign changes, and move under a sign error in Phi or a structure never multiplied by Phi.
- The suite's header comment and `delivery-status.md` are remeasured.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `verification-integrity`: the requirement on judging a multimodal rotation changes to decide the same solution by the loadings and compare its frame fields at `factorRotationTol`; a new requirement compares rotation-invariant quantities strictly.

## Impact

- `stats/factor_analysis_test.go` only: `assertFactorAnalysisMatchesR` gains a scoring parameter and the layered comparison, the tolerances and their comments change, and the unit test is added. No library code changes, so no changelog entry.
- The strict suite is measured before (691 failing leaves) and after. Failures move from position-dependent fields to invariants where the fitted model itself differs, which is where extraction drift already fails.
