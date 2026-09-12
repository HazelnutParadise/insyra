# Proposal: oblimin-honours-its-start

## Why

`FaRotations` runs every rotation from the start it is handed, except `oblimin`, which builds its own identity matrix on every pass and ignores the start. `Rotation.Restarts: 20` therefore performs the same computation twenty times and returns the first result — the parameter claims a search that never happens, and `Docs/stats.md` promises one ("the number of starting points the rotation is run from; the solution with the best criterion value wins"). Oblimin is the rotation `DefaultFactorAnalysisOptions()` selects, so it is the method this matters most for. Found while fixing #373 (`orthogonal-rotation-starts`), which made every start orthogonal — the starts oblimin refuses are now legitimate ones. Recorded as an `AGENTS.md` follow-up on 2026-09-12 because the comment on the identity start calls it deliberate, "better SPSS compatibility than random starts", so the fork was SPSS parity against an honest `Restarts`.

Measured before deciding, on `buildSyntheticTable(60, 6, syntheticGen3Factor)` with ML extraction (the table `TestRestartsParameter` uses), comparing the identity-only oblimin against one that uses its starts. The criterion value `f` is oblimin's own objective, lower is better; the loading difference is taken after matching column order and sign:

| Case | Identity only | Using the starts | max\|ΔL\| |
| --- | --- | --- | --- |
| 3 factors, `Restarts` 1 / 2 / 5 | 0.001264147695 | bit-identical | 0 |
| 3 factors, `Restarts` 20 | 0.001264147695 | 0.001264147691 | 2.3e-6 |
| 3 factors, `Delta` 0.5 / −0.5, `Restarts` 20 | — | lower by ≤ 1.0e-11 | ≤ 2.2e-6 |
| **4 factors, `Restarts` 5** | 0.044366801178 | **0.000941589675** | **0.96** |
| 5 factors, `Restarts` 20 | 0.009916640074 | 0.009751770499 | 0.039 |
| the 12 generated datasets in `factor_analysis_test.go`, `Restarts` 20 | — | lower by 4e-12 to 1.1e-10 | 7e-7 to 1.2e-5 |

Three things follow. `Restarts: 1` is bit-identical because `buildStarts` returns the identity alone and premultiplying by it is a no-op, so the default path and every R parity test (all at `Restarts: 1`) cannot move. On a well-specified model the search finds the same minimum from every start, which is what GPArotation's own local-minima vignette reports for oblimin (200 of 200 random starts to one solution). On an over-factored model — four factors fitted to three-factor data, which the default Kaiser count produces routinely — the search finds a basin the identity start never reaches, with a criterion 47 times lower. Today's code spends the time anyway: at five factors `Restarts: 20` takes 202 ms to return the 9.9 ms answer.

The SPSS claim is true and beside the point. IBM's Algorithms document (FACTOR, Oblique Rotations, Initializations) says "The factor correlation matrix C is initialized to Im" and offers only δ; SPSS is single-start. That behaviour lives at `Restarts: 1`, which is the default and which this change leaves bit-identical. A caller who asks for twenty starts is asking for something SPSS does not do, for any method.

What the two R references actually do settles the fork. `psych::faRotations` (psych 2.6.5, the function this file is named after) builds `starts[[1]] <- diag(ncol(loadings))` followed by QR-orthogonalised random matrices, and calls `GPArotation::oblimin(loadings, Tmat = initial)` — oblimin receives its start like every other method, with no special case. `GPArotation::oblimin()` itself takes `randomStarts` and its engine (`.GPA_RS_engine`, 2026.8-2) keeps the start with the strictly smallest criterion value. psych 2.6.5 also changed `fa()`'s `n.rotations` default from 1 to 20. Neither reference applies a threshold before a random start may displace the identity, so none is added here.

## What Changes

- **BREAKING**: `oblimin` rotates from each start in the list, exactly as the other nine methods do: `pre = L·start`, rotate from the identity, compose `rotmat = start·R`. `FactorAnalysis` with `Rotation.Method: FactorRotationOblimin` and `Rotation.Restarts > 1` can now return a different solution than at `Restarts: 1` — the one with the lowest criterion value among the starts that converged, which is what `Restarts` has documented since `orthogonal-rotation-starts`.
- The identity-start special case and its SPSS comment are removed, and so is the `rotateLower == "oblimin"` branch that skipped composing the rotation matrix with the start.
- Results at `Restarts: 1` are bit-identical: the only start is the identity.
- Two regression tests pin the behaviour. In `stats/internal/fa`, oblimin at `gamma = 0` and quartimin are the same criterion, so for the same starts they must return the same criterion value — they did not, at `Restarts ≥ 5` on an over-factored fixture. In `stats`, the over-factored case above must reach the lower criterion at `Restarts: 5`.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that every method runs from the start it is given, so `Restarts` means the same search for all of them.

## Impact

- `stats/internal/fa/psych_faRotations.go`: the `"oblimin"` arm of the switch in `FaRotations` and the `finalRot` composition below it. No signature changes.
- `stats.FactorAnalysis` callers who set `Rotation.Restarts > 1` with oblimin see a result that may differ from `Restarts: 1`. On the twelve generated datasets the difference is at the 1e-6 level; on an over-factored model it can be a different basin. On the measured four-factor case that basin has a factor correlation of 0.905 and a smallest `Phi` eigenvalue of 0.094 — a lower criterion value is the search doing what it was asked, not a promise that the solution is the one an analyst wants. Quartimin, GeominQ and BentlerQ already behave this way; this change makes oblimin consistent with them, and this caveat belongs in the documentation of `Restarts`, not in a per-method exception.
- `Restarts: 1`, `DefaultFactorAnalysisOptions()`, and the R parity tests in `factor_analysis_test.go` are unaffected: measured bit-identical on eight configurations, and structurally so.
- The wasted work goes away: `Restarts: 20` for oblimin now buys twenty distinct starts instead of twenty identical runs.
- Both `CHANGELOG` files gain an entry under `stats`, appended after the `orthogonal-rotation-starts` entries it extends. `api-review.md` gains a row. The `AGENTS.md` follow-up is deleted and `delivery-status.md`'s decision delta is updated.
