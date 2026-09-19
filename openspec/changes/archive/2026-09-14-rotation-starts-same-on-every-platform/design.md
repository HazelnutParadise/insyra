## Context

`buildStarts` in `stats/internal/fa/psych_faRotations.go` returns the identity, the Varimax rotation matrix, then QR-orthogonalised random matrices drawn from `rand.New(rand.NewSource(seedFromMatrix(baseLoadings)))`. `seedFromMatrix` folds `math.Float64bits` of every loading into the seed, so a change in any bit of any loading gives an unrelated stream. The loadings reach `FaRotations` from extraction (ML through L-BFGS-B, MINRES, PAF, PCA), whose last digits depend on gonum's assembly kernels and on how the compiler contracts floating-point expressions. `ENG.md` already records that bit parity is a property of the platform, not of the code.

## Goals / Non-Goals

**Goals:**
- The same `FactorAnalysis` call draws the same random starts on every platform, so a multi-start search reaches the same basin wherever it runs.
- `Restarts: 1` stays bit-identical.

**Non-Goals:**
- Bit-identical extraction across architectures. It would mean disabling gonum's assembly and controlling contraction throughout the extraction path, which costs speed for a property `ENG.md` already rules out, and a difference at 1e-8 in the loadings is not the defect.
- A `Seed` option on `FactorRotationOptions`. `KMeansOptions` has one, but the defect needs none, and adding one is an API decision of its own.
- Revisiting how many starts the default runs or how the winner is chosen.

## Decisions

- **A fixed seed, not a platform-stable function of the data.** Rounding the loadings before hashing still has boundaries a 2e-8 difference can straddle, and a seed from the dimensions carries nothing a random orthogonal start needs. Neither reference ties the starts to the data: `psych::faRotations` draws them from R's session RNG unseeded, so they change from run to run, and `GPArotation::Random.Start` does the same. A fixed seed keeps the reproducibility the data hash gave for a single platform and extends it to every platform.
- **Seed value 1**, the value `0.4` chose before measuring which start reaches the lower basin, kept here so the two lines draw the same starts.
- **Quartimin, not oblimin, in the regression test.** The defect is about which starts are drawn, so the test needs a method that uses them. Oblimin on this line rotates from its own identity whatever it is handed, which is a separate gap recorded in `AGENTS.md`; quartimin is the same criterion at `Delta` 0 and does use its starts.
- **Regression test on one machine.** The defect shows as two platforms disagreeing, which a single test process cannot observe directly. Its cause can be observed: set one entry of `overFactoredStructure()` to the value amd64's extraction produces (a change of 2.0e-8) and require `buildStarts` to return bit-identical random starts, and the search at `Restarts: 5` the same criterion.

## Risks / Trade-offs

- [The random matrices are different ones, on every platform] → Where the criterion has several minima the chosen solution can change. Measured on this line over the 20 datasets and four extractions: up to 2.15 per loading for simplimax and 1.63 for geominQ at `Restarts >= 2`, and nothing at all at `Restarts: 1`. The model invariant stays at 1e-15, so what moves is which minimum is found, not whether the answer is a rotation. The strict R parity suite runs at the default and reports the identical 1,842 failing leaves before and after.
- [A start that sits on a basin boundary could still split platforms by floating-point noise] → Possible for any start, and unrelated to the seed. After the change the only arm64/amd64 disagreements left at `Restarts >= 2` are the eight that already exist at `Restarts: 1`. CI on three operating systems is the ongoing check.
