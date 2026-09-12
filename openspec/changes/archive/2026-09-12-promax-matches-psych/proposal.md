# Proposal: promax-matches-psych

## Why

Promax was the largest remainder of the strict R parity suite after `factor-parity-compares-what-it-means`: 1,028 of its 1,672 failing leaves, on 11 of 16 datasets, at `Restarts: 1` and 20 alike. Stepping psych 2.6.5's `kaiser(loadings, rotate = "Promax")` and ours side by side on the suite's ten-row table (`factorAnalysisRows`, MINRES, two factors) found the unrotated loadings identical to ten digits, the Kaiser weighting identical, and the first divergence inside `Promax` itself: psych now runs `GPArotation::Varimax(x, Tmat = diag(k))` as the pre-rotation ("replaced with GPArotation Varimax 5/9/26" in `psych/R/Promax.R`, shipped in 2.6.5), where it used `stats::varimax(x)`; our `fa.Promax` mirrors the older code through the `stats::varimax` port with Kaiser normalisation.

On data with a clear varimax optimum the two agree to the convergence tolerance. On this table every weighted row is (±1, ≈0), the varimax criterion is nearly flat in the rotation angle, `stats::varimax` stops at [0.796, −0.605] and `GPArotation::Varimax` at [0.744, −0.668] — and Promax raises that to the fourth power, so loading[0,0] came out 0.700 against psych's 0.625 and `Phi[0,1]` −0.878 against −0.881. Our own gradient projection Varimax stops where GPArotation's does, 1e-4 apart, so following psych's choice of pre-rotation is enough to follow its Promax.

## What Changes

- **BREAKING**: `fa.Promax` pre-rotates with `Varimax` (gradient projection from the identity, no Kaiser normalisation, `eps = 1e-5`) instead of the `stats::varimax` port. `FactorAnalysis` with `Rotation.Method: FactorRotationPromax` returns psych 2.6.5's solution; on data with a clear varimax optimum the change is at the convergence tolerance, on a nearly flat one it is the difference between the two algorithms' stopping points, up to 0.08 in a loading on the parity table.
- A test pins psych 2.6.5's `Promax(weighted, m = 4)` on that table to 5e-4.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that Promax's pre-rotation is the same Varimax psych uses.

## Impact

- `stats/internal/fa/psych_Promax.go`, step 1. `KaiserVarimaxWithRotationMatrix` stays for `VarimaxAlgorithm: VarimaxKaiser`, which is still psych's `varimax` (`stats::varimax`).
- Every Promax result moves, most by the convergence tolerance. Both changelogs carry a **BREAKING** entry.
- The strict parity suite's Promax leaves drop from 1,028 to what is measured after the change and written into the `factorParityTol` comment; the `AGENTS.md` follow-up is deleted.
