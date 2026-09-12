# Proposal: orthogonal-rotation-starts

## Why

[#373](https://github.com/HazelnutParadise/insyra/issues/373). `stats.FactorAnalysis` with `Rotation.Restarts > 1` returns loadings that no longer describe the same model. It is a wrong answer through a public statistical API, with no error and no warning.

A rotation criterion is minimised over a constrained set: `T'T = I` for the orthogonal family, `diag(T'T) = I` for the oblique one. The gradient projection algorithms (Jennrich 2001, 2002) project every step back onto that set, so an iterate never leaves it — provided the start is on it. A start off the set does not give a worse rotation, it gives a different model, which is why the invariant breaks.

`FaRotations` adds three heuristic starts whenever `restarts > 1`, and two of them, Promax and TargetRot, are not orthogonal. Whichever start wins on criterion value decides the answer, so whether the result is right depends on the data.

Measured on 6 variables and 3 factors, `max|L·L' − Lu·Lu'|` for the orthogonal family and `max|L·Φ·L' − Lu·Lu'|` for the oblique one:

| Method | Restarts = 1 | Restarts ≥ 2 |
| --- | --- | --- |
| Varimax | 3.3e-16 | **1.0035** |
| Quartimax | 6.7e-16 | **1.0035** |
| BentlerT | 6.7e-16 | **1.0035** |
| GeominT | 4.4e-16 | 4.4e-16 |
| BentlerQ (oblique) | 4.4e-16 | **0.7627** |

The last row is the part the issue does not cover: **the defect is not confined to orthogonal rotations.** `FaRotations` premultiplies the loadings by the start (`pre = L·start`) instead of passing it to the algorithm as its `Tmat`. For an orthogonal start the two are the same computation — `GPForth` opens with `L = A·Tmat` — but for an oblique start they are not, because the oblique parameterisation is `Λ = A·(T')⁻¹`. Either way the invariant needs `S·S' = I`. `GPFoblq` already normalises the columns of the start it is given and its comment names Promax as the case it is defending against, but `FaRotations` never passes a start to it, so that guard has never run.

A second defect sits in the same loop. The candidate map carries loadings, rotmat, Phi and the criterion value, but not the convergence flag, so `best` never has one and `fa.Rotate` falls back to `converged = true`. `FactorAnalysis.RotationConverged` is therefore always true. Measured: GeominQ logs `Oblique rotation did not converge after 1001 iterations` and reports `RotationConverged: true` in the same call. Picking the minimum criterion value across starts without regard to whether those runs converged is the same rule being broken twice.

## What Changes

- Every start given to a rotation is orthogonal. The list is the identity matrix, the Varimax rotation matrix, and QR-orthogonalised random matrices — which is what `GPArotation::Random.Start` produces for both families. The Promax and TargetRot starts are dropped: after orthogonalisation neither retains anything of the method it came from, so it would be an arbitrary orthogonal matrix that a random start already supplies.
- A start is verified orthogonal before use. One that is not is skipped and replaced, so a rotation helper that fails to converge while building a start cannot contaminate the result.
- `Restarts` becomes the number of starts. It used to bound only the random ones while the three heuristics were added unconditionally, so `Restarts: 2` ran four starts.
- Every rotation wrapper propagates its convergence flag, the candidate carries it, and the best-of-restarts rule prefers a converged solution: the minimum criterion value among converged starts, or — if none converged — the minimum overall, reported as not converged.
- `fa.Rotate` and `FactorAnalysis.RotationConverged` report what actually happened.

Not changed, and not the fix: threading the start into `GPForth`/`GPFoblq` as `Tmat` rather than premultiplying. With orthogonal starts the two are equivalent — put `T = S·R` and `(T')⁻¹ = S·(R')⁻¹`, so `L·(T')⁻¹ = L·S·(R')⁻¹` and `T'T = R'S'SR = R'R`, the same criterion value at the same point of the same manifold. It would only admit start matrices that `GPArotation` does not use, and this file mirrors `GPArotation`.

Also left alone: `oblimin` ignores the start entirely and rotates from the identity on every pass, so `Restarts` costs it N identical computations and buys nothing. The comment says the identity start is deliberate, for SPSS parity. Changing it would move oblimin results, which is a separate decision and is recorded as a follow-up rather than taken here.

## Capabilities

### New Capabilities

- `stats-factor-rotation`: what a rotation must preserve, what a restart may start from, and what the convergence flag means.

### Modified Capabilities

(none)

## Impact

- `Rotation.Restarts = 1`, the default from `DefaultFactorAnalysisOptions()`, produces the same numbers as before. The heuristic start block never ran for it.
- `Restarts > 1` with Varimax, Quartimax, BentlerT or BentlerQ changes. What it returned was not a rotation of the fitted model.
- `Restarts > 1` now runs `Restarts` starts rather than `Restarts + 2`, so a result that depended on a heuristic start winning can move even where the invariant held.
- `RotationConverged` can now be false. It was a constant true, so nothing could have depended on a false. Measured before the fix: all nine iterative methods reported `converged = true` after a single iteration at a tolerance of 1e-12.
- `TestRotate_RestartsBreakOrthogonality` is deleted. It was written to pin the defect and to fail once it was fixed, naming its own replacement, which is `rotation_starts_test.go`.
- `TestRestartsParameter` ran GeominQ alone, so it checked the oblique invariant and never the orthogonal one. It now covers all ten methods, and fails on four of them before the fix.
- `TestRotationConvergedFlag` tried to force a non-convergence with `opt.MaxIter = 1`, which governs extraction and never reaches the rotation, and wrote the assertion as a `t.Logf` so it passed either way. The negative case moves to the `fa` package, where the knob exists.
- `fa.TargetRot` is left in place though nothing but a test calls it now. It is part of the `GPArotation` transliteration and a target rotation would need it.
