## Context

`FaRotations` in `stats/internal/fa/psych_faRotations.go` loops over the starts `buildStarts` returns (identity, Varimax rotation matrix, QR-orthogonalised random matrices) and, for nine of ten methods, premultiplies the loadings by the start before rotating from the identity, then composes the rotation matrix as `start·R`. The `"oblimin"` arm builds a fresh identity, passes the raw loadings to `GPFoblq`, and the composition below skips the start for oblimin. Both halves have to change together, or the loadings and the rotation matrix stop agreeing.

`preferCandidate` already implements the selection rule: a converged solution beats one that did not converge, and among equals the lower criterion value wins. Nothing in this change touches it.

## Goals / Non-Goals

**Goals:**
- Oblimin runs from each start exactly as quartimin does. At `gamma = 0` the two are the same criterion, which gives a test that cannot pass by accident.
- `Restarts: 1` stays bit-identical.

**Non-Goals:**
- A threshold before a random start may displace the identity ("only switch if clearly better"). Considered because the observed movement on well-specified models is 1e-6 in the loadings for a 4e-12 gain in criterion, and rejected: neither reference does it — `GPArotation`'s engine keeps the strictly smallest `f`, `psych::faRotations` ranks by hyperplane count — and it would change the selection rule for all ten methods, which is a different decision.
- Following psych 2.6.5's new `n.rotations = 20` default. That would move the default output; it is a separate decision and is noted in `delivery-status.md`.
- Reconciling our selection rule (minimum criterion among converged starts, from `orthogonal-rotation-starts`) with psych's (hyperplane count, then complexity, then fit). The file is named after psych's function but selects the way `GPArotation`'s engine does; recorded as a follow-up.
- Threading the start into `GPFoblq` as `Tmat` instead of premultiplying. `orthogonal-rotation-starts` already established the two are the same optimisation for an orthogonal start.

## Decisions

- **Use the starts (option 1) rather than refuse `Restarts > 1` for oblimin (option 2).** Statistically, `Restarts` is a multi-start minimisation of a non-convex criterion, and a start that is not used is not a start. Both R references pass the start to oblimin: `psych::faRotations` calls `GPArotation::oblimin(loadings, Tmat = initial)` with no per-method exception, and `GPArotation::oblimin()` exposes `randomStarts`. Option 2 would have meant changing `Docs/stats.md`'s documented meaning of `Restarts` and carving one method out of a parameter that is uniform across the other nine. The SPSS single start is preserved where an SPSS comparison is made, at `Restarts: 1`.
- **Same composition as the other oblique methods.** `pre = L·start`, `GPFoblq(pre, I, …)`, `rotmat = start·R`. With an orthogonal start `Phi = T'T` is unchanged by the composition, so the oblique invariant `L·Φ·L' = Lu·Lu'` holds — `TestRestartsParameter` and `TestRotationPreservesTheModelAcrossRestarts` check it at up to 20 starts.
- **Tests first.** Two tests fail before the fix: in `fa`, oblimin and quartimin disagree at `Restarts ≥ 5` on a pinned over-factored fixture (the four-factor ML loadings of the synthetic table, where the identity start stops in a basin with `f = 0.0444` and a random start reaches `f = 0.00094`); in `stats`, the same case through the public API. Both are green after.

## Risks / Trade-offs

- [A caller with `Restarts > 1` and oblimin gets a different solution] → It is the solution the documentation already promised. On the twelve generated datasets the movement is ≤ 1.2e-5 in the loadings; only an over-factored model changes basin. Called out as **BREAKING** in both changelogs, appended to the `orthogonal-rotation-starts` entries it extends, in the same unreleased version.
- [The lower-criterion basin is substantively worse on an over-factored model] → On the measured case `Phi` has an off-diagonal of 0.905 and a smallest eigenvalue of 0.094. This is a property of multi-start oblique rotation, already true of quartimin, GeominQ and BentlerQ, and belongs in the documentation of `Restarts` rather than in a per-method exception. The default `Restarts: 1` does not do it.
- [Another session is editing this working tree] → `delivery-status.md`, `AGENTS.md`, both changelogs and `api-review.md` are shared with `parquet-foreign-column-types`; each is re-read immediately before it is edited.
