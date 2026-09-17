# Proposal: simplimax-follows-gparotation

## Why

Simplimax in `stats` minimises a different criterion from the one psych 2.6.5 minimises whenever three or more factors are fitted, or when two squared loadings are equal. psych calls `GPArotation::simplimax(A, Tmat = Tmat)` without a `k`. The wrapper then sets `k = nrow(A) * (ncol(A) - 1)`, and `vgQ.simplimax` adds up exactly the `k` smallest squared loadings, in the order `order()` gives them, which breaks ties by column-major position. The port fixes `k` at the number of variables, and adds every squared loading at or below the `k`-th smallest, so a tie adds more than `k` of them.

Measured on 2026-09-14 against `GPArotation:::vgQ.simplimax` 2026.8.2, at the identity:

| Loadings | GPArotation's default `k` | GPArotation's criterion | The port's criterion |
| --- | --- | --- | --- |
| 6×2, the parity suite's `two_blocks` (MINRES) | 6 | 0.0062690 | the same |
| 6×3, the pattern the gradient tests use, with repeated values | 12 | 0.245 | 0.125 (at `k = 6`, counting ties; GPArotation at `k = 6` gives 0.0575) |
| 6×4, the over-factored fixture | 18 | 0.75799 | 0.0035756 (GPArotation at `k = 6` gives the same) |

Rotated from the identity, GPArotation stops at a criterion of 0.021799 on the 6×3 matrix and 0.094133 on the 6×4 one, while the port drives its own criterion to 3e-9 and 1e-10. Those are different rotations, not the same rotation to a different precision. `fa.Criterion`, which the strict parity suite uses to decide whether our solution and psych's are the same minimum, also uses the port's `k`, so it judged psych's solutions by a criterion psych did not minimise. Found while pinning `rotations-use-gparotation-bb`'s reference values, and recorded as an `AGENTS.md` follow-up the same day. The strict suite currently has 26 failing simplimax leaves out of 709.

## What Changes

- **BREAKING**: the simplimax criterion adds up exactly the `k` smallest squared loadings, with `k = number of variables × (number of factors − 1)`, and breaks ties by column-major position as R does. With two factors and no equal squared loadings, the criterion and the results are unchanged.
- The rotation, the choice among multi-start starts and `fa.Criterion` all evaluate that one criterion.
- The internal `Simplimax` wrapper loses its `k` parameter, which it accepted and never passed to the criterion.
- Tests pin `vgQ.simplimax`'s criterion and gradient from GPArotation 2026.8.2 on loadings with and without ties, at the default and at an explicit `k`. The simplimax reference cases in `gparotation_bb_test.go` are regenerated at GPArotation's default `k`, including the 6×3 case that was left out for this defect. Both fail on the current criterion.
- The `AGENTS.md` follow-up is deleted.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that Simplimax minimises GPArotation's criterion, with its default `k` and its tie order.

## Impact

- `stats/internal/fa/GPArotation_vgQ_simplimax.go`, `stats/internal/fa/GPArotation_GPFoblq.go`, `stats/internal/fa/criterion.go`, `stats/internal/fa/psych_faRotations.go`. No exported API changes.
- `stats.FactorAnalysis` with `FactorRotationSimplimax` and three or more factors, or with equal squared loadings: a different solution, the one psych reports. The strict R parity suite is measured before (709 failing leaves, 26 of them simplimax) and after.
- Both `CHANGELOG` files gain a **BREAKING** entry under `stats`. `Docs/stats.md` states the criterion. `api-review.md` gains row ST-15. `delivery-status.md` gains a milestone. `stats/factor_analysis_test.go`'s header comment is remeasured.
