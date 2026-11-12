# 0025 — Factor rotation follows GPArotation's engine and psych's defaults

**Status:** accepted
**Date:** 2026-09-14

## Context
`stats`' factor analysis is checked against psych 2.6.5 and GPArotation 2026.8.2. Between 2026-09-12 and 2026-09-17 several defects in the rotation and in its parity suite came out together: oblimin ignored the start it was handed, oblique rotations started from non-orthogonal matrices, the random starts were seeded from the loadings, and the gradient projection stepped the way GPArotation's old function does.

## Decision
- Gradient projection rotations follow GPArotation's current default algorithm, `"bb"`, with no option to choose the old step: nothing exposed the algorithm and no measured case needed it (`rotations-use-gparotation-bb`).
- `Restarts` defaults to 20, psych's `n.rotations`, and `Restarts: 1` stays bit-identical to a single start for those who want SPSS or GPArotation's behaviour. Every method, oblimin included, uses its starts (`default-restarts-follow-psych`, `oblimin-honours-its-start`).
- Every start is orthogonal, for the oblique family as well as the orthogonal one, which is what GPArotation uses (`orthogonal-rotation-starts`).
- The random starts come from a fixed seed, not from the loadings (`rotation-starts-same-on-every-platform`).
- Among starts, a converged solution beats one that did not converge; among converged ones the lowest criterion value wins. When none converged the best is returned and reported as not converged.
- Parity against psych is held strict by what a rotation cannot change (L·Phi·L', S·L', W·L'). The frame fields of one solution share one tolerance, `factorRotationTol` = 1e-2, measured from GPArotation's own same-minimum spread (`factor-parity-compares-invariants`).

## Options rejected
- psych's hyperplane count as the selection rule. The criterion defines the rotation, GPArotation's engine and `fungible::faMain` both select its minimum, and on three fixtures with twenty starts the two rules chose the same basin every time.
- A threshold before a random start may displace the identity start. Neither GPArotation nor `psych::faRotations` has one.
- Threading the start into `GPForth`/`GPFoblq` as `Tmat` instead of premultiplying the loadings. With an orthogonal start the two are the same optimisation.
- Seeding the starts from a platform-stable function of the data. Rounding still has boundaries a 2e-8 difference can cross, and neither reference ties its starts to the data.
- Per-combination tolerances from psych's seed-to-seed drift. psych keeps the identity start most of the time, so its drift understates what the criterion allows.
- Comparing iteration counts against GPArotation. Rounding carried through the step size moves where a long run stops on every platform, R's included; the reference test compares the first six iterations step by step and the end by convergence, criterion and loadings.

## Consequences
psych 2.6.5's own tie-break among starts has a defect that our port does not share; reporting it upstream waits for the owner (`AGENTS.md` Follow-ups).
