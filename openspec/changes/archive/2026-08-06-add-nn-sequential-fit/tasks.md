# Tasks: add-nn-sequential-fit

## 1. Surface

- [x] 1.1 Define `FitConfig` (Epochs, BatchSize, Seed, NoShuffle, Optimizer, Loss, ValX, ValY, Progress, Quiet), the optimizer/loss selector types over the existing tape methods (SGD, SGDMomentum, Adam, AdamW; CrossEntropy, MSE, BCEWithLogits), `FitEpoch` progress payload, and `FitResult` (per-epoch train/val losses, elapsed). Validation errors name the missing or invalid field; absent loss or optimizer is refused, never defaulted.

## 2. The loop

- [x] 2.1 Implement `Sequential.Fit`: seeded `rng.Perm` shuffling per epoch (skipped under NoShuffle), batch slicing, forward through the layer surface, selected loss, `Backward`, selected optimizer step — dispatching to the existing tape methods with nothing numeric reimplemented.
- [x] 2.2 Per-epoch reporting: mean train loss accumulated in float64 host-side, optional validation loss via the Predict path (training-only layers structurally excluded), one info line per epoch through the root logger, callback and Quiet semantics per the spec.

## 3. Verification

- [x] 3.1 The digit-for-digit gate: a test runs the documented hand-written loop and an equivalently configured Fit under the same seed and asserts identical per-epoch loss sequences — the ENG.md sugar-changes-nothing proof extended to Fit. This test must be shown to fail when Fit's batch walk is perturbed, then pass. Verification: reversing the Fit permutation produced a failing parity run (`Fit losses = [0.72311526536941528 0.6682032744089762 0.62545039256413781 0.59934226671854651]`, hand loop `[0.72542091210683191 0.67152901490529382 0.63157473007837928 0.59327304363250732]`); reverting to `rng.Perm` passed.
- [x] 3.2 Determinism: two identical Fit runs produce identical parameters; NoShuffle preserves input order; refused configs error as specified.
- [x] 3.3 Progress contract tests with captured logger output: N epochs → N info lines, Quiet+callback composition, validation loss presence.
- [x] 3.4 The ungated micro-convergence test gains a Fit twin; the gated MNIST run gains a Fit arm reaching the M21 accuracy numbers (record the run). The Fit micro-convergence twin passes. Acceptance run (M3, local dataset): the Fit arm passes with Fit's own pinned trajectory — losses 0.347310 / 0.165883, accuracy 95.47%, clearing the M21 bar. The original assertion of M21's exact digits was corrected during acceptance: M21 drew shuffle permutations from the post-initialization stream, while Fit's documented Seed contract uses its own stream, so both are deterministic but necessarily different.
- [x] 3.5 Full `go test ./nn/...` passes; race detector on Fit with a concurrent-validation... no — Fit is single-goroutine by design; run the standard suite and the existing race-tagged tests unchanged. Final standard and race runs both pass.

## 4. Docs, changelog, skills, bookkeeping

- [x] 4.1 Rewrite the `Docs/nn.md` training section around Fit with the hand-written tape loop retained as the advanced path; state the determinism contract and the progress line.
- [x] 4.2 Changelog entries under `## Unreleased` in both `CHANGELOG.md` and `CHANGELOG_TW.md` (`### nn` heading); sync `skills/insyra/` training guidance.
- [x] 4.3 `delivery-status.md`: milestone-style record with the digit-for-digit verification signal; note learning-rate schedules and early stopping as recorded follow-ups, deliberately excluded from v1.
- [x] 4.4 `openspec validate add-nn-sequential-fit --strict` passes.
