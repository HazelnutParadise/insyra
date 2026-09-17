## Context

`assertFactorAnalysisMatchesR` in `stats/factor_analysis_test.go` aligns our factors to psych's from the loadings, then compares extraction fields at `factorParityTol` (2e-5) and factor-frame fields at `frameTol`. `frameTol` rises to `factorRotationTol` only for a gradient projection rotation whose loadings differ by more than 2e-5 and whose criterion is within a relative 1e-4 of psych's; a lower criterion skips every frame field, a higher one fails. The references come from `crosslang_baseline.R`, seeded since `factor-baselines-reproducible`. See proposal.md for the measurements and the research behind this change.

## Goals / Non-Goals

**Goals:**
- Strictness that does not depend on where along a flat minimum either side stopped, and that still catches an inconsistent frame.
- One rule for the same solution: every frame field at one tolerance, as GPArotation and EFAtools do.

**Non-Goals:**
- Changing the criterion margin (1e-8 + 1e-4·|f|). It now only has to say "no worse", and tightening it needs its own measurement.
- Invariants for Anderson-Rubin scores, which use the symmetric inverse square root of L'U⁻²RU⁻²L and are not rotation-invariant for an oblique rotation. They stay under the frame comparison.
- Changing any library code.

## Decisions

- **Loadings first, criterion second.** A criterion value cannot separate two minima that share it to six digits (Nguyen & Waller 2023), while different minima differ in the loadings by 0.1 and more. Deciding by the loadings also means a tighter criterion margin later can only add failures, never silent skips.
- **The invariants run before the verdict.** A different, lower minimum still has to be the same fitted model, and Promax, which has no criterion, gets the same check.
- **`factorRotationTol` = 1e-2 rather than per-combination tolerances.** Measuring psych's drift per combination across seeds was tried first: psych's hyperplane selection keeps the identity start most of the time, so its seed-to-seed drift (1.4e-3 in Phi on `three_blocks` with MINRES and bentlerQ) understates what the same criterion allows, which GPArotation from random starts shows directly (6.0e-3 on the same loadings). One constant tied to the flattest measured case is simpler and, with the invariants strict, gives up no precision.
- **W·L' at its own tolerance.** It equals R⁻¹S·L' for regression scores, so the extraction's own 6e-6 difference arrives multiplied by R⁻¹; 1e-4 is twice the 4.5e-5 measured on `narrow_plus_group` with ML.
- **A scoring parameter instead of inferring the method.** The model does not record its scoring method, and the six call sites already hold it.

## Risks / Trade-offs

- [A formula error in structure, Phi or scores smaller than 1e-2 would pass the frame comparison] → The invariants catch structure and Phi errors at 2e-5, and W·L' catches regression and Bartlett weight errors at 1e-4; Anderson-Rubin scores are the one field left at 1e-2.
- [New invariant fields fail where the extraction already drifts] → Expected on `near_collinear`, where the extraction fields fail today; they are reported under their own names so the cause stays visible.
- [The criterion margin stays relative 1e-4] → With the loadings deciding, it only bounds "no worse"; tightening it is left as a separate measurement.
