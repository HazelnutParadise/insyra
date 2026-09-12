## Context

`FaRotations` picks among starts with `preferCandidate` (converged first, then lowest criterion value), and `buildStarts` produces the identity, a Varimax rotation matrix, and QR-orthogonalised random matrices. `stats.DefaultFactorAnalysisOptions()` sets `Restarts: 1`, and `normalizeFactorAnalysisOptions` maps a zero to that default. The three R implementations compared are GPArotation 2026.8-2 (`R/GPFRS.R`), psych 2.6.5 (`R/faRotate.r`, `man/fa.Rd`) and fungible (`R/faMain.R`), all read from their CRAN sources.

## Goals / Non-Goals

**Goals:**
- Decide, and record, which reference the selection rule and the default follow.
- Keep the default path's cost proportional to the search it runs.

**Non-Goals:**
- Adopting psych's hyperplane-count selection. Rejected below.
- Dropping the informed Varimax start to match the references exactly (none of the three uses one). It was added deliberately in `orthogonal-rotation-starts`; at the search's own tolerance it costs 0.03 ms, so there is nothing to gain by removing it.
- Changing the per-start non-convergence warning. Pre-existing for explicit `Restarts > 1`; recorded as a follow-up.
- Raising `maxit` from 1000 to GPArotation's new default of 2000.

## Decisions

- **Lowest criterion value among converged starts, not hyperplane count.** The criterion defines the rotation, so its minimum is the estimator; `GPArotation`'s engine and `fungible::faMain` both select the minimum, and fungible is the source psych credits for the multi-start technique. psych's hyperplane count is a readability heuristic on a `0.15` cutoff. Measured on all three fixtures, twenty starts each, the two rules choose the same basin every time, including the over-factored one where the lower basin has both the lower criterion and the higher hyperplane count — so psych's rule does not even buy the protection one might hope for. Alternative considered: hyperplane count with a `Hyper` option. Rejected as an extra option that changes nothing measured and departs from two of the three references.
- **Default 20, psych's number.** psych is the package's reference implementation (`ENG.md`; the baseline script calls `psych::fa` with its defaults), its author changed the default on a real local-minimum example, and fungible defaults to 10 for the same reason. Alternative considered: keep 1, matching SPSS and `GPArotation`. Rejected: those are a procedure and an engine, not the analysis front-end this package mirrors, and a single start is the estimator only when the criterion happens to be unimodal. Alternative considered: 10, fungible's number. Rejected in favour of matching the reference.
- **Informed start at the rotation's tolerance.** `buildStarts` takes `eps` and `maxIter` and passes them to `Varimax`. A start is a point on the manifold, verified orthogonal before use; converging it to 1e-8 buys nothing and on five of the twenty datasets cost 76 to 86 ms per call, which is what made the default look expensive. Alternative considered: keep `1e-8 / 5000` and accept the cost. Rejected on the measurement.

## Risks / Trade-offs

- [Every default `FactorAnalysis` result moves] → by at most 1.2e-5 on the twenty datasets in the suite, inside the 2e-5 R parity tolerance, and always within the same basin; called out as **BREAKING** in both changelogs with `Restarts: 1` as the way back.
- [Default cost rises] → 1.5 to 9 ms on the suite's datasets instead of 0.2 to 0.7 ms, once the informed start stops hitting its cap. Measured per oblimin start: 0.07 to 0.6 ms, 240 of 240 converged.
- [A criterion with local minima logs a warning per unconverged start] → unchanged from what explicit `Restarts: 20` did; follow-up.
- [Results at `Restarts > 1` for every method shift at the noise level] → the informed start is now computed to `eps` rather than `1e-8`; the model invariant tests bound the effect at 1e-10.
