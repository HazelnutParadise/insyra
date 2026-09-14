# Proposal: rotation-starts-same-on-every-platform

## Why

The `Test` workflow on `0.4` has been red on `ubuntu-latest` and `windows-latest` since 3234297 (`chore(openspec): archive oblimin-honours-its-start`, run 34691376316, 2026-09-12), while `macOS-latest` and an Apple M3 pass. Every red run fails the same assertion in `TestObliminRestartsSearchTheCriterion`: four factors fitted to the three-factor synthetic table, rotated by oblimin, must reach a lower criterion at `Restarts: 5` than at `Restarts: 1`. On amd64 both are 0.044367. `GOARCH=amd64 go test` under Rosetta reproduces it on the M3.

The extra starts are used on amd64. The search draws different ones. Logged start by start at `Restarts: 8` (start 0 is the identity, start 1 the Varimax solution, the rest random):

| Build | Unrotated ML loading `[1,1]` | Seed | Starts reaching f = 0.00094 |
| --- | --- | --- | --- |
| amd64 | 0.39670959426082741 | −988248667554548133 | 6, 7 |
| amd64, `-tags noasm` | 0.39670957504473991 | −1508811493714070953 | 2, 5 |
| arm64, with or without `noasm` | 0.39670957381899075 | 2547506732042636995 | 4, 5 |

Swept over `Restarts` 1 to 20, amd64 first reaches the lower basin at 7 and arm64 at 5. Every start converged on both.

**Root cause.** `buildStarts` seeds the random starts with `seedFromMatrix(baseLoadings)`, a hash of every bit of the unrotated loadings. Extraction does not reproduce those bits across architectures: the ML loadings above differ in the eighth decimal. Part of that is gonum's amd64 assembly kernels, since `noasm` moves the amd64 value, and part remains with gonum in pure Go on both; which operation diverges first was not isolated, because the fix does not depend on it. A difference of 2e-8 is harmless to the loadings. Hashed into a seed, it selects an unrelated set of random matrices, so the same `FactorAnalysis` call on the same data returns a solution in a different basin on Linux than on a Mac. On this fixture that is the 0.0444 solution against the 0.00094 one, which `oblimin-honours-its-start` measured 0.96 apart in the loadings.

So the code is wrong and the test is right. The test's claim, that the extra starts reach a basin the identity start does not, holds on every platform. What does not hold is that the reached basin is the same one for the same call. Its internal twin, `TestObliminRunsFromTheStartItIsGiven`, passes on every platform only because it pins the arm64 loadings as literals, so both architectures hash identical bits. The number five in the test is the first draw of one particular random stream that lands in the lower basin; once the stream no longer depends on the loadings' bits, it is the same draw everywhere, and the test needs no change.

## What Changes

- **BREAKING** for `Rotation.Restarts > 1`: the random starts are drawn from a fixed seed instead of a hash of the loadings, and `seedFromMatrix` is removed. On a criterion with more than one minimum the chosen solution can differ from before on every platform, because the random matrices are different ones.
- Measured with a fixed seed on amd64, amd64 `noasm`, arm64 and arm64 `noasm`: each of eight starts lands in the same basin on all four builds, the criterion values agree to 1e-10, and the lower basin is first reached at start 3.
- Measured with the strict R parity suite (psych 2.6.5, GPArotation 2026.8.2) on an Apple M3: 873 failing leaves with the data-derived seed, 922 with seed 1, and 879 to 906 with seeds 2 to 5. Nearly all of the difference is `rotation_converged` (50 with the old seed, 82 to 99 with a fixed one) and the fields that depend on it, for quartimin, oblimin and geominQ on the two ten-row tables. Fitted with two factors those tables are nearly rank one, and a rotation there converges within 1000 iterations only from a start a few degrees from the solution: 7 to 31 of 400 random starts for quartimin, 0 to 3 for geominQ. Whether one of eighteen random starts lands there is luck, and a data-derived seed is not luckier: over 60 seeds of each kind, hashes of slightly perturbed loadings and small integers gave at least one converging start about equally often (14 and 15 of 60 on `two_blocks`, 40 and 46 on `missing_rows`). The verdicts on the quality of the minimum found barely move: "a worse minimum than R's" covers three simplimax combinations with the old seed and three to five with a fixed one, four with seed 1. Seed 1 stays: choosing the seed with the lowest count would fit the suite's luck, not improve the method. The same suite under `GOARCH=amd64` with seed 1 reports 928 leaves and agrees with arm64 on every convergence and minimum verdict: the same 99 `rotation_converged` leaves and the same 12 simplimax ones. The 46 leaves that differ between the two are scores, score coefficients, structure and `Phi` entries at the 2e-5 tolerance, where extraction's floating-point difference decides which side of the tolerance a value falls.
- `Restarts: 1` is unaffected, because the identity is its only start and no random stream is drawn. The informed Varimax start is unaffected.
- A regression test in `stats/internal/fa` sets one entry of the pinned over-factored loadings to its amd64 value and requires the same random starts and the same search result. It fails before the change on both architectures.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `stats-factor-rotation`: adds the requirement that the random starts do not depend on the loadings' floating-point bits, so a multi-start result is the same on every platform.

## Impact

- `stats/internal/fa/psych_faRotations.go`: `buildStarts` and `seedFromMatrix`. No signature changes.
- `stats.FactorAnalysis` with `Rotation.Restarts > 1`, which is the default since `default-restarts-follow-psych`: the same call now reaches the same basin on amd64 and arm64. Where several minima exist, or where only a few starts converge, the solution may change from the unreleased `0.4` behaviour. On the strict R parity suite that is 873 to 922 failing leaves, within the 879 to 906 other fixed seeds give.
- CI: `TestObliminRestartsSearchTheCriterion` passes on all three operating systems without being changed.
- Both `CHANGELOG` files gain an entry at the end of `stats`. `Docs/stats.md` and `skills/insyra/references/stats.md` say the random starts come from a fixed seed. `api-review.md` gains row ST-13. `delivery-status.md` gains a milestone and a decision.

## Correction (2026-09-14)

The cause given above for the `rotation_converged` movement is wrong. Quartimin and geominQ rarely converged on the two ten-row tables because the port stepped the way GPArotation's old function does, not because the data are nearly one-dimensional. On the same loadings GPArotation's default `"bb"` algorithm converges from 400 of 400 random starts, so with that algorithm the random-start seed makes no difference on those tables, and the parity count did not move with the seed because convergence there is luck. The fixed seed and its reasoning stand. `rotations-use-gparotation-bb` ports the algorithm.
