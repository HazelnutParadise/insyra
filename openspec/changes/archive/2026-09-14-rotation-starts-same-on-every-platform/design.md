## Context

`buildStarts` in `stats/internal/fa/psych_faRotations.go` returns the identity, the Varimax rotation matrix, then QR-orthogonalised random matrices drawn from `rand.New(rand.NewSource(seedFromMatrix(baseLoadings)))`. `seedFromMatrix` folds `math.Float64bits` of every loading into the seed, so a change in any bit of any loading gives an unrelated stream. The loadings reach `FaRotations` from extraction (ML through L-BFGS-B, MINRES, PAF, PCA), whose last digits depend on gonum's assembly kernels and on how the compiler contracts floating-point expressions. `ENG.md` already records that bit parity is a property of the platform, not of the code.

`seedFromMatrix` dates from the first factor analysis implementation (ca7cf68, 2025-10-05). Nothing in the specs, the docs or the commit history says why the seed follows the data.

## Goals / Non-Goals

**Goals:**
- The same `FactorAnalysis` call draws the same random starts on every platform, so a multi-start search reaches the same basin wherever it runs.
- `Restarts: 1` stays bit-identical.
- `TestObliminRestartsSearchTheCriterion` passes on amd64 and arm64 unchanged.

**Non-Goals:**
- Bit-identical extraction across architectures. It would mean disabling gonum's assembly and controlling contraction throughout the extraction path, which costs speed for a property `ENG.md` already rules out, and a difference at 1e-8 in the loadings is not the defect.
- A `Seed` option on `FactorRotationOptions`. `KMeansOptions` has one, but the defect needs none, and adding one is an API decision of its own.
- Revisiting how many starts the default runs or how the winner is chosen. Both were decided in `default-restarts-follow-psych`.

## Decisions

- **A fixed seed, not a platform-stable function of the data.** Rounding the loadings before hashing still has boundaries a 2e-8 difference can straddle, and a seed from the dimensions carries nothing a random orthogonal start needs. Neither reference ties the starts to the data: `psych::faRotations` draws them from R's session RNG unseeded, so they change from run to run, and `GPArotation::Random.Start` does the same. A fixed seed keeps the reproducibility the data hash gave for a single platform and extends it to every platform.
- **Seed value 1**, chosen before measuring which start reaches the lower basin on the fixture, so the value is not tuned to the test.
- **The test stays as it is.** Its premise, that the extra starts are used and reach a basin the identity does not, is true and is what `Docs/stats.md` promises. Its count of five is a property of the random stream, and that stream is now the same on every platform. Raising the count to 20 would have hidden the cross-platform difference instead of removing it.
- **Regression test on one machine.** The defect shows as two platforms disagreeing, which a single test process cannot observe directly. Its cause can be observed: set one entry of `overFactoredStructure()` to the value amd64's extraction produces (a change of 2.0e-8) and require `buildStarts` to return bit-identical random starts, and the search at `Restarts: 5` the same criterion. Before the fix both assertions fail on both architectures.

## Risks / Trade-offs

- [The random matrices are different ones, on every platform] → Where the criterion has several minima, or converges from only a few starts, the chosen solution can change from the unreleased `0.4` behaviour. It is marked **BREAKING**. Measured on the strict R parity suite: 873 failing leaves before, 922 after, 879 to 906 for seeds 2 to 5. The movement is `rotation_converged` on two ten-row tables where a rotation converges only from a start a few degrees from the solution, so it is the luck of which starts are drawn. Hash-derived seeds were checked against integer seeds over 60 of each and were not luckier, and the worse-minimum verdicts moved only from three simplimax combinations to four.
- [Tuning the seed to the parity count] → Rejected. Seed 5 would report 879 leaves instead of 922, but that is fitting a fixed seed to one test corpus and to psych's own unseeded draws. The seed stays at the value chosen before any measurement.
- [A start that sits on a basin boundary could still split platforms by floating-point noise] → Possible for any start, and unrelated to the seed. On the fixture all eight starts agree across four builds. CI on three operating systems is the ongoing check.
- [Other sessions push to `0.4`] → The change is made on its own branch and rebased onto `origin/0.4` before pushing. `delivery-status.md`, both changelogs and `api-review.md` are re-read right before they are edited.
