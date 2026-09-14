# Proposal: rotations-use-gparotation-bb

## Why

Every gradient projection rotation in `stats` still steps the way GPArotation did before its default changed. That covers Varimax, Quartimax, GeominT, BentlerT, Quartimin, Oblimin, GeominQ, BentlerQ, Simplimax, the informed Varimax start of a multi-start search, and the Varimax inside Promax. GPArotation 2026.8.2 runs every criterion with `algorithm = "bb"`, `fwindow = 10` and `maxit = 2000` unless told otherwise, and psych 2.6.5 calls it without naming an algorithm, an iteration cap or a tolerance. The R answers the strict parity suite compares against were computed that way.

Measured on 2026-09-14 on the unrotated loadings of the parity suite's `two_blocks` (MINRES) and `missing_rows` (PCA) tables, two factors, `eps = 1e-5`, `maxit = 1000`, 400 random starts, counting the starts that converged:

| | quartimin | geomin |
| --- | --- | --- |
| GPArotation `"bb"` | 400 and 400 | 400 and 400 |
| GPArotation's retained old function `GPFoblq.legacy` | 6 and 13 | 0 and 2 |
| `stats/internal/fa` today | 7 and 31 | 0 and 3 |

With `"bb"` every start, the identity included, reaches the same minimum in 30 to 180 iterations. The old step crawls along the nearly flat criterion and stops at the cap. From the same 120 starts, our `GPFoblq` matches `GPArotation:::GPFoblq.legacy` at every iteration (criterion to 1e-9, step size exactly, the same convergence flag), so the port is faithful. It follows an algorithm the reference has moved away from.

This is the strict suite's `rotation_converged` row, 99 failing leaves on 2026-09-13. `rotation-starts-same-on-every-platform` explained that row, and the way the suite's count moved with the random-start seed, as the data being nearly one-dimensional and convergence being luck. That explanation is wrong: when almost no start converges, which starts are drawn decides whether one does, and with the reference algorithm the seed makes no difference on these tables. The wrong cause is written in `stats/factor_analysis_test.go`'s header comment, in `delivery-status.md`, and in that change's archived proposal and design. `Docs/stats.md` also says the rotation uses R's `maxit = 1000`, which is only the old function's default.

## What Changes

- **BREAKING**: the gradient projection step follows GPArotation 2026.8.2's default. The first iteration doubles the step size as before. From the second on, the step size is estimated from how the rotation matrix and the projected gradient changed over the previous iteration, and held between 1e-10 and 20. A trial step is accepted when it improves on the largest criterion value of the last ten iterations rather than on the current one, halving up to ten times as before. An oblique rotation matrix that cannot be inverted is inverted through its singular value decomposition, as R's `safe_inverse` does, instead of the trial being skipped without halving the step.
- **BREAKING**: the default iteration cap of a gradient projection rotation becomes 2000, R's default. It applies to the rotation `FactorAnalysis` runs, to the informed Varimax start, and to Promax's Varimax pre-rotation. The `stats::varimax` pre-rotation used for principal components Promax keeps its 1000, because `stats::varimax` does.
- Rotations that already converged reach the same minimum. Rotations on a nearly flat criterion now converge, and a multi-start search can choose a different solution because more of its starts finish.
- Tests pin reference values from GPArotation 2026.8.2 for every criterion: the iteration count, the final criterion value and the loadings, from the identity and from a fixed random start. They fail on the current step.
- The explanations named in Why are corrected. The archived proposal and design gain a correction note rather than being rewritten.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that gradient projection rotations step and stop the way GPArotation 2026.8.2 does by default, and raises the iteration cap in the informed-start requirement's scenario from 1000 to 2000.

## Impact

- `stats/internal/fa/GPArotation_GPFoblq.go`, `stats/internal/fa/GPArotation_GPForth.go`: the step loop and the matrix inversion. No signature changes.
- `stats/internal/fa/psych_faRotations.go`, `stats/internal/fa/fa.go`, `stats/internal/fa/psych_Promax.go`, `stats/factor_analysis.go`: the default iteration cap and the comments that state it.
- `stats.FactorAnalysis` with any gradient projection rotation or Promax after a factor extraction: loadings can move where the old step stopped short of the minimum, and `RotationConverged` can change from `false` to `true`. The strict R parity suite is measured before (922 failing leaves on 2026-09-13) and after.
- Both `CHANGELOG` files gain a **BREAKING** entry under `stats`. `Docs/stats.md` and `skills/insyra/references/stats.md` state the algorithm and the cap. `api-review.md` gains row ST-14. `delivery-status.md` gains a milestone and a decision, and its `rotation-starts-same-on-every-platform` entry has its cause corrected.
