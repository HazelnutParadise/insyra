# Proposal: rotation-starts-same-on-every-platform

## Why

`buildStarts` seeds the random starts of a multi-start rotation with `seedFromMatrix(baseLoadings)`, a hash of every bit of the unrotated loadings. Extraction does not reproduce those bits across architectures: ML extraction of four factors from the 60-row synthetic table in `stats/verify_more_test.go` gives loading `[1,1]` = 0.39670957381899075 on arm64 and 0.39670959426082741 on amd64. A difference of 2e-8 is harmless to the loadings. Hashed into a seed, it selects an unrelated set of random matrices, so the same `FactorAnalysis` call on the same data returns a solution in a different basin on one architecture than on another.

Measured on this line over the 20 datasets of `stats/factor_analysis_test.go`, all four extractions and all ten rotation methods, comparing a native arm64 run against the same run under `GOARCH=amd64`: 197 of the 2400 `Restarts >= 2` combinations disagree by more than 1e-5, the worst of them by 2.1 per loading (simplimax) and 1.63 (geominQ). At `Restarts: 1` only 8 of 800 exceed 1e-5, all of them at 1.5e-5 or below, which is extraction noise and not a different solution.

On `0.4` the same defect made the `Test` workflow red on `ubuntu-latest` and `windows-latest` since 3234297 while macOS passed, through `TestObliminRestartsSearchTheCriterion`. That test belongs to `oblimin-honours-its-start` (6c4fce1), which is not on this line, so here the defect is latent rather than failing CI — but it is the same defect, and it is exposed the moment anyone sets `Restarts > 1` on two machines.

`seedFromMatrix` dates from the first factor analysis implementation (ca7cf68, 2025-10-05). Nothing in the specs, the docs or the commit history says why the seed follows the data.

## What Changes

- The random starts are drawn from a fixed seed instead of a hash of the loadings, and `seedFromMatrix` is removed.
- Measured after the change: the arm64/amd64 disagreement at `Restarts >= 2` falls from 197 of 2400 to 24 of 2400, and those 24 are the same 8 dataset/extraction/rotation combinations that already disagree at `Restarts: 1`, repeated at `Restarts` 2, 5 and 20, at the same 1.5e-5 or below.
- `Restarts: 1` is unaffected, because the identity is its only start and no random stream is drawn: bit-identical over all 800 combinations. The informed Varimax start is unaffected.
- Results at `Restarts >= 2` move on every platform, because the random matrices are different ones — measured at up to 2.15 per loading for simplimax and 1.63 for geominQ. The model invariant `max|L·Φ·L' − Lu·Lu'|` stays at 1e-15 throughout, so what moves is which minimum is found, not whether the answer is a rotation.
- A regression test in `stats/internal/fa` sets one entry of the pinned over-factored loadings to its amd64 value and requires the same random starts and the same search result. It fails before the change on both architectures.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that the random starts do not depend on the loadings' floating-point bits, so a multi-start result is the same on every platform.

## Impact

- `stats/internal/fa/psych_faRotations.go`: `buildStarts` and `seedFromMatrix`. No signature changes.
- `stats.FactorAnalysis` with `Rotation.Restarts > 1`. The default on this line is 1, so the default path does not move.
- The strict R parity suite runs at the default and is therefore unchanged: 1,842 failing leaves before and after, the same leaves with the same numbers.
- Both `CHANGELOG` files gain an entry at the end of `stats`. `Docs/stats.md` and `skills/insyra/references/stats.md` say the random starts come from a fixed seed.

## Backport to dev (0.3.x)

This is the dev copy of `0.4`'s `rotation-starts-same-on-every-platform` (dcc220c, archived in 44fc5c7). It is taken here because the defect is in the code this line now carries, and because a rotation that answers differently on Linux than on a Mac is wrong by any reading.

Adapted:
- The regression test rotates with **quartimin** rather than oblimin. At `Delta` 0 they are the same criterion, but oblimin on this line builds its own identity start and ignores the ones it is handed, so it cannot show a basin difference. Quartimin on the same fixture reaches f = 0.0444 from the identity and the Varimax start and f = 0.00094 from the first random start, so `Restarts: 3` is where it leaves the identity's basin. Before the fix the test fails on its start-equality half on both architectures; its criterion half happens to pass on this fixture, because both hashes reached the lower basin by five starts.
- The `overFactoredStructure` comment describes quartimin and says what oblimin does instead.
- 0.4's parity movement (873 to 922 failing leaves) has no counterpart here: 0.4's default is twenty starts, this line's is one.
- No **BREAKING** mark, as nothing in this line's `## Unreleased` carries one.

Left on 0.4: `api-review.md` row ST-13 and `delivery-status.md`.
