# Proposal: factor-baselines-reproducible

## Why

The strict R parity suite compares `FactorAnalysis` against baselines that `stats/testdata/crosslang_baseline.R` produces, and those baselines are not reproducible. `psych::fa` rotates from `n.rotations = 20` starting points, 19 of them drawn from R's random number generator, and the script never seeds it. Measured on 2026-09-17 by running the script three times with the same payload (the `two_blocks` table, ML extraction, oblimin, regression scores, two factors): the loadings agreed to 3.5e-6, but Phi differed by 2.3e-5 and 5.3e-5 between runs, structure by up to 5.3e-5 and scores by up to 6.4e-5. The suite compares those fields at 2e-5, so the reference fails its own tolerance against itself, and whether a leaf passes depends on the R session that happened to build the cache.

`psych::principal` defaults to a single start and draws no random numbers, so the PCA baselines are unaffected.

Reading psych 2.6.5's `faRotations` for this also found a defect in how it chooses among starts that tie on hyperplane count: the second tie-break returns a position within the tied subset instead of the start's own index, and the third is computed without being assigned. With starts 1 and 3 tied and start 3 the less complex, the code selects start 2. That is psych's code; the port selects by criterion value and does not share it. It means a seeded baseline is reproducible, not necessarily the start psych meant to choose, and it is recorded where the baseline is built.

## What Changes

- The factor analysis baseline seeds R's random number generator with a fixed value before calling `psych::fa` and `psych::principal`, so the same payload yields byte-identical output in every session.
- A test runs the script twice on that payload, bypassing the cache, and requires identical output. It fails on the current script.
- The script's comment records why the seed is there and the `faRotations` tie-break defect.
- The strict parity suite is remeasured against regenerated baselines.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `verification-integrity`: adds the requirement that a reference baseline which draws random numbers seeds its generator, so a cached reference does not depend on the session that built it.

## Impact

- `stats/testdata/crosslang_baseline.R` (`factor_analysis_stats`) and a new test in `stats/factor_analysis_test.go`. No library code changes, so no changelog entry.
- Every R baseline cache entry is regenerated once, because the cache key includes the script's content.
- `delivery-status.md` gains a milestone.
