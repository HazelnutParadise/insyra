# Proposal: test-low-coverage-packages

## Why

TS-18 of #309 listed eight packages below 50%. `test-untested-packages` covered three (`isr`, `internal/utils`, and `internal/algorithms`, which had already risen above the line during the review's own batches). This is the rest, or as much of it as is worth doing.

What was actually uncovered turned out to be more interesting than the percentages suggested:

- **`cli/repl`'s whole uncovered half was `completer.go`** — tab completion, which a user touches on every line of the REPL. None of it needed a terminal: `Do` takes a line and a cursor position and returns what is left to type.
- **`stats/internal/fa`'s was the entire GPArotation family** — `GPForth`, `GPFoblq` and the seven `vgQ*` criterion functions. These are transliterations of R's GPArotation and had never been run by a test. A wrong gradient there does not crash; it walks the rotation to the wrong place while still reporting convergence.
- **`datafetch`'s is mostly the Google Maps crawler**, whose removal is #249 and still undecided, plus the live HTTP paths. What is left is two helpers that decide what a caller sees.

## What Changes

Tests only.

| Package | Before | After | What is now covered |
| --- | --- | --- | --- |
| `cli/repl` | 39.1% | 72.1% | the completion contract (suffixes plus the typed length), command / variable / file completion, the path-command table, sorting, and the directory-separator suffix |
| `stats/internal/fa` | 22.2% | 34.3% | every criterion's analytic gradient against a numerical derivative, orthogonality and communality invariance for `GPForth`, unit column norms for `GPFoblq`, `SymmetricEigenDescendingDsyevr`'s eigenpairs, `KaiserVarimaxWithRotationMatrix`, `Rotate` |
| `datafetch` | 44.5% | 46.6% | `RateLimitError`'s two message forms and its `errors.Is` contract, `normalizeDateColumns`'s last-token heuristic, `sleepBackoff` |

R is not available here, so the rotation tests check properties that hold by definition rather than reference numbers: a gradient must match the derivative of its own objective, an orthogonal rotation must return an orthogonal matrix and leave each variable's communality alone, an oblique one must keep unit column lengths. Comparing against R belongs with the other reference verifications (#302/#303).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `test-suite-integrity`: gains the rule that a numerical routine ported from another implementation is tested against the properties its mathematics guarantees when the original is not available to compare with.

## Impact

- Three new test files. No non-test change.
- `api-review.md` TS-18; `delivery-status.md`.
- **Two of TS-18's eight are deliberately left.** `ml/mltest` (40.7%) is a conformance-test helper: its coverage comes from the packages that use it, and testing a test harness against itself adds little. `accel/knnbridge` (21.1%) is low because its tests are gated on `INSYRA_ACCEL_GPU_TESTS`, which is #303's subject, not a coverage problem.
- **The rotation tests found a defect and pinned it rather than fixing it**: `fa.Rotate` with more than one restart returns a non-orthogonal matrix for an orthogonal rotation, because `FaRotations` seeds the search with Promax's and TargetRot's rotation matrices, which are oblique. Reproduced at the public API: `stats.FactorAnalysis` with Quartimax and `Restarts: 5` returns loadings whose `L·L'` differs from `Lu·Lu'` by 0.287, where `Restarts: 1` gives 7.8e-16. The default is `Restarts: 1`, so the default path is sound. Reported separately; `TestRotate_RestartsBreakOrthogonality` fails when it is fixed, which is when that test should be replaced by the communality check.
