## Context

`assertFactorAnalysisMatchesR` in `stats/factor_analysis_test.go` compares fifteen fields of a `FactorModel` against a psych baseline element by element at `factorParityTol = 2e-5`. Baselines come from `runRBaseline`, cached under `testdata/baseline_cache/<exe>/` by `baselineCacheKey(scriptContent, exe, method, payload)`. The rotation criteria live in `stats/internal/fa` as unexported `vgQ*` functions.

## Goals / Non-Goals

**Goals:**
- Fail the suite only on differences a correct implementation is obliged not to have.
- Make a cached baseline say which reference produced it.

**Non-Goals:**
- Loosening `factorParityTol`. Extraction-level drift on ill-conditioned data stays visible.
- Seeding psych's random starts or pinning R package versions in CI. The versions are recorded, not fixed.
- Reworking `TestFieldDiffAllFailingCombos`, a diagnostic report.

## Decisions

- **One alignment from the loadings, applied everywhere.** Considered aligning each field independently; rejected because a rotation matrix that is a permutation of its own loadings would then pass. The loadings define the frame; `Phi` and score covariance transform on both sides (`D·P'·Φ·P·D`), column-indexed matrices on their columns, proportion vectors by permutation with the cumulative recomputed. The one exception is the rotation matrix: measured on 2026-09-13, the loadings' alignment left quartimax, geominT, bentlerT and bentlerQ differing by a column swap or a sign, because psych's `rot.mat` is the matrix before its own factor sorting and sign standardisation. It is compared up to its own permutation and sign — the equivalence class a rotation matrix is defined in — which is weaker than the loadings' frame but is the only frame both sides share.
- **Criterion, not location, for the GPA family.** Considered listing only simplimax, geomin and bentler as multimodal; rejected because oblimin and quartimin are multimodal on over-factored data (measured in `oblimin-honours-its-start`) and a hand-picked list would be wrong the day a new dataset joins. Every method whose `FaRotations` arm reports `f` uses the criterion fallback; promax is closed-form and keeps element-wise comparison. The tolerance on the criterion is `1e-8 + 1e-6·|f_r|`.
- **Versions in the key rather than in the file.** Considered writing the versions into the cached JSON and checking on read; rejected because a key miss regenerates silently while a file check needs a policy for mismatch. The probe runs once per test process per interpreter and its output string is hashed; a probe that fails hashes its error, so a broken toolchain does not alias a working one.

## Risks / Trade-offs

- [The first strict run regenerates every baseline] → about 18 minutes for R on this machine, once per toolchain change.
- [A criterion pass hides a real difference] → only when our criterion is at or below R's, which is the definition of at least as good; the difference is logged with both values so it is not invisible.
- [Promax stays red on two datasets] → it is not a criterion minimisation; recorded in the `factorParityTol` comment as what remains.
