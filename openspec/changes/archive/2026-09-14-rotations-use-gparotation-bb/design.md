## Context

`GPForth` (orthogonal) and `GPFoblq` (oblique) in `stats/internal/fa` are the only gradient projection engines. Every GPA rotation wrapper in `psych_faRotations.go` calls one of them, `buildStarts` calls the Varimax wrapper for its informed start, and `psych_Promax.go` calls the same wrapper for the psych 2.6.5 pre-rotation. Both engines transliterate GPArotation's old function: the step size doubles at the top of every iteration, and a trial is accepted only when it improves on the current criterion value. `GPFoblq` additionally skips a trial whose matrix fails to invert without halving the step.

GPArotation 2026.8.2 keeps the same outer structure in `GPForth` and `GPFoblq` and changes three things under `algorithm = "bb"` (its default for every wrapper, reached from psych through `.GPA_RS_engine` with `eps = 1e-5`, `maxit = 2000`, `fwindow = 10`): the step size from the second iteration on, the acceptance target, and the oblique inverse. Its convergence test, `s < eps` on the Frobenius norm of the projected gradient, is unchanged. See proposal.md for the measurements.

The iteration cap is set in four places: the fallback in `FaRotations` (1000), the nil-options default in `fa.Rotate` (1000), the literal in `psych_Promax.go`'s Varimax call (1000), and comments in `fa.go` and `stats/factor_analysis.go` that state it. `KaiserVarimaxWithRotationMatrix` is `stats::varimax`, not a GPA engine, and keeps its own 1000.

## Goals / Non-Goals

**Goals:**
- Both engines produce GPArotation 2026.8.2's default trajectory: the same steps from the first iteration, the same convergence flag, and the same criterion value and loadings to floating-point accumulation.
- One cap, 2000, wherever a GPA rotation runs without an explicit one.

**Non-Goals:**
- An option to select the old step. Nothing in the library exposes the algorithm, no measured case needs the old step, and R users who want it name `algorithm = "legacy"` in R. It can be added later without changing this behaviour.
- `algorithm = "cayley"`. It is opt-in in GPArotation and orthogonal only.
- GPArotation's non-convergence diagnostic for factor correlations above 0.85. It changes a message, not a result, and was offered to the owner separately.
- A caller-supplied seed for the random starts.

## Decisions

- **Replace the step rather than add a second one.** The engine gains the three differences and keeps its structure line for line against the R source, so a future diff against GPArotation stays readable. The alternative, a mode flag with both steps, doubles the code for a path nothing calls.
- **Port `safe_inverse` exactly.** `solve` first; on failure the pseudo-inverse from the singular value decomposition with singular values at or below `sqrt(.Machine$double.eps)` treated as zero. The Go port's current `continue` on a failed inversion leaves the step size unchanged and moves to the next trial, which is a different trajectory from R's even under the old step. R records a singularity flag it does not act on; the port does not add a warning for it.
- **A criterion that fails ends the rotation from that start, as in R.** The old port skipped a trial whose criterion returned an error (bentler's criterion inverts a matrix) and went on to the next trial. GPArotation has no such branch: the error stops `GPFoblq`. The port now returns the error, and a multi-start search drops that start the way it already drops a start whose rotation fails.
- **Iteration and table semantics stay as R's.** The table holds one row per evaluated iteration, `iter` from 0, with the step size in force at the start of that iteration. The step estimate reads the previous iteration's rotation matrix and projected gradient, and is skipped (the step size carried over unchanged) when the projected gradient did not change, as R does when its denominator is zero.
- **The cap follows the reference everywhere a GPA rotation runs.** R's informed start in psych is the rotation itself, and psych's Promax calls `GPArotation::Varimax` with defaults, so 2000 in all three places. `stats::varimax` keeps 1000.
- **The first six iterations are compared step by step, the end state by result.** Measured after the port: the criterion and the step size agree with GPArotation to twelve digits at the start of every case, and the gap grows past 1e-9 only after 25 to 50 iterations, because the step size is a ratio of small differences near convergence and carries rounding forward. Runs of 11 and 80 iterations end on the same iteration as R, while longer ones end a few iterations apart (140 against 142, 892 against 651), and amd64 and arm64 differ from each other in the same way. Comparing the first six iterations to 1e-10 is what catches a step that is ported wrong: a logic difference shows at the first iteration it affects. The end state is compared by convergence flag, criterion value (1e-8 relative) and loadings (1e-4, the largest measured gap being 4.5e-5 on amd64), not by iteration count.
- **Reference values pinned as literals, generated by GPArotation 2026.8.2.** This matches `promax_psych_test.go`: no R at test time, the exact R call written in the test's comment. Cases cover every criterion from the identity on four loading matrices (the two ten-row tables, the three-factor pattern the gradient tests use, and the over-factored four-factor fixture), plus a fixed random start on the flat and multimodal ones, plus normalised Varimax. Simplimax is pinned at `k = nrow(A)`, the criterion the port implements, so the test checks the step and not the criterion; GPArotation's own default differs, see Risks.
- **Tests first.** The reference test and a test that the cap defaults to 2000 fail on the current engines before any engine change.

## Risks / Trade-offs

- [Results move for rotations that stopped at the cap] → Marked **BREAKING** in both changelogs, landing in the same unreleased 0.4 as `default-restarts-follow-psych`, so users meet one change to multi-start results. The strict R parity suite is measured before and after.
- [Rounding carries forward through the step size, so a long run's iteration count differs between R, amd64 and arm64] → Measured, and the test contract follows from it (Decisions). The reference test runs under `GOARCH=amd64` as well as on arm64.
- [Simplimax's criterion does not match GPArotation 2026.8.2 for three or more factors] → Found while building the reference: GPArotation's `simplimax` wrapper, which psych calls without a `k`, uses `k = nrow(A) * (ncol(A) - 1)` and exactly the `k` smallest squared loadings, while the port uses `k = nrow(A)` and every value at or below the `k`-th. The two agree for two factors. It is a separate defect in the criterion, not the step, so it is recorded as a follow-up for the owner rather than folded into this change.
- [Existing tests assume the old step does not converge] → Rechecked after the port: `go test ./stats/...` passed on arm64 and amd64 with no existing test changed, because the tests that exercise non-convergence force it with a tolerance or a cap no step can meet.
- [The over-factored oblimin test depends on which starts reach the lower basin] → Measured in R under `"bb"`: the identity still stops at f = 0.04437 and 18 of 40 random starts reach f = 0.00094, so the premise holds. After the port, `TestObliminRestartsSearchTheCriterion` still reaches the lower basin within five starts on arm64 and amd64.
