# Proposal: factor-parity-compares-what-it-means

## Why

The strict factor-analysis parity suite (`INSYRA_STRICT_FACTOR_R_PARITY=1`) measures things a correct implementation is not obliged to reproduce, and trusts a cache that cannot tell which R produced it. Run on 2026-09-12 against baselines regenerated under psych 2.6.5 and GPArotation 2026.8.2, it failed 5,589 leaf sub-tests before `oblimin-honours-its-start` and 5,334 after, while the comment at `factorParityTol` describes ~595 on three adversarial datasets. Three causes, all in the suite rather than in the library:

- **Rotation matrices, loadings and `Phi` are compared element by element.** A factor solution is defined up to the order and sign of its factors. Both sides sort factors and standardise signs, but psych's `rot.mat` is the matrix before that step, so a column swap or a sign flip shows as a difference of 2.0 — 1,311 of the failing leaves are `rotation_matrix`.
- **A multimodal criterion is judged by which minimum R reached.** `simplimax`, `geomin` and `bentler` have several local minima (GPArotation's own vignette: simplimax found 16 on one dataset). psych now runs 20 unseeded random starts and picks by hyperplane count; we run 20 seeded starts and pick the lowest criterion. Where the two land in different minima the suite fails every rotation-dependent field, although the criterion value is the only thing that says which answer is better — and on the two simplimax cases where the new default differs from psych, ours is lower (0.1445 against 0.1599, 0.1657 against 0.2270).
- **The baseline cache is keyed on script and payload only.** The 2026-08-01 cache answered for whatever psych produced it until moved aside; nothing in the key or the file records the R or Python package versions, so a stale reference passes as current without anyone knowing which R the numbers came from.

## What Changes

- `assertFactorAnalysisMatchesR` aligns our factors to R's once, from the loadings — the column permutation and signs that minimise `max|L_go − L_r|` — and applies that alignment to every factor-indexed field: loadings, structure, `Phi` and score covariance (both sides), scores, score coefficients, the rotation matrix, and the explained and cumulative proportions.
- For a gradient-projection criterion, aligned loadings that still differ beyond the tolerance are judged by the criterion: `fa.Criterion` (new, exported from the internal package) evaluates the method's own `vgQ` at both loading matrices, with Varimax's Kaiser normalisation. A solution whose criterion is at or below R's passes with the difference logged as a different minimum; one above R's fails naming both values. Promax has no criterion and stays element-wise.
- The baseline cache key includes the reference toolchain's versions — for `Rscript` the R version and the psych, GPArotation and jsonlite versions; for `python` the interpreter and numpy, scipy and statsmodels versions — probed once per test process. Every existing cache entry is therefore regenerated on the next run, and a cache produced by one psych can no longer answer for another.
- The comment at `factorParityTol` states what the suite reports after these changes, against the R it was run against, instead of a figure from 2026-05-03.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `verification-integrity`: adds the requirements that a factor solution is compared up to factor order and sign, that a multimodal rotation is judged by its criterion value, and that a cached reference baseline is bound to the toolchain versions that produced it.

## Impact

- Test code only. `fa.Criterion` is new exported API on `stats/internal/fa`, reachable only inside `stats`; no library behaviour changes and there is no changelog entry.
- The strict suite's failure count drops from 5,334 to whatever the three fixes leave, which is measured and written into the `factorParityTol` comment. What remains is extraction-level drift on ill-conditioned data and Promax, which have their own explanations there.
- The first strict run after this change regenerates every R and Python baseline: about 18 minutes for the R side on this machine.
- `TestFieldDiffAllFailingCombos` keeps its own element-wise diagnostics; it is a report, not a gate.
