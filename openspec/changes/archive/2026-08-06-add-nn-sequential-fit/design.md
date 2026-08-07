# Design: add-nn-sequential-fit

## Context

Training today is a caller-owned loop: seed an RNG, permute indices, slice batches, run tape ops, call an optimizer method (`tape.SGD/SGDMomentum/Adam/AdamW`), repeat. ENG.md fixes two relevant facts: parameters materialize seeded at `Build` in layer order, and the Sequential surface carries a "sugar changes nothing" proof — the Sequential MNIST run must reproduce the hand-written tape run's loss curve to the last digit. M21 deliberately kept dataset loading and seeding test-side; a Fit method must add a front door without smuggling in ambient randomness.

## Goals / Non-Goals

**Goals:**

- One call trains a `Sequential` with visible progress, and its parameter trajectory is bit-identical to the equivalent hand-written loop.
- Determinism by construction: every random choice (shuffle order) flows from an explicit seed in the config; no global RNG, no time-derived state.
- The existing optimizers and losses are selected, not reimplemented.

**Non-Goals:**

- No learning-rate schedules in v1 (`StepLR`/cosine exist as tape-level tools; composing them into Fit is a recorded follow-up, not a silent inclusion).
- No dataset/loader API, no augmentation, no early stopping, no checkpointing — v1 is the loop, not the ecosystem.
- No `ml` protocol integration in this change; Fit speaks Tensors.
- No changes to gradient, optimizer, or loss semantics.

## Decisions

1. **Config over variadics.** `FitConfig{Epochs, BatchSize, Seed, NoShuffle, Optimizer, Loss, ValX, ValY, Progress, Quiet}` — explicit fields, zero-value-safe where a zero is meaningful, validated where it is not (Epochs and BatchSize must be positive; Loss and Optimizer must be set — a training call with an unstated objective is refused, not defaulted, because a silently chosen loss is a wrong-model generator).
2. **Optimizer and loss are small typed selectors over existing tape methods** — e.g. `OptimizerSpec`/`LossSpec` values (`SGD{Rate}`, `Adam{Rate}`, `AdamW{Rate, WeightDecay}`, `SGDMomentum{Rate, Momentum}`; `CrossEntropy`, `MSE`, `BCEWithLogits`). Fit dispatches to `tape.Adam(...)` etc.; nothing numeric is reimplemented, so every existing PyTorch-parity proof carries over untouched.
3. **Shuffling is seeded and index-based.** `math/rand` with the config seed produces the per-epoch permutation exactly the way the documented loop does (`rng.Perm`), so a hand loop and Fit given the same seed walk the same batches. `NoShuffle` preserves input order for debugging and curriculum cases. Seed is a plain int64 with 0 as a valid, honest seed — no "0 means random" trapdoor.
4. **Progress is a per-epoch info line by default, a callback when asked, silence when told.** The line carries epoch k/N, mean train loss, val loss if validation tensors were given, elapsed, and rows/s, through the root logger under its level control. `Progress func(FitEpoch)` receives the same numbers; `Quiet` suppresses the default line without touching the callback. Per-batch logging stays out of v1 — the epoch line already proves liveness, which is the problem being solved.
5. **The acceptance gate is inherited, not invented.** The change's proof obligation: a Fit call configured to match the documented MNIST hand loop (same seed, same batches, same optimizer) reproduces that loop's loss sequence digit-for-digit, extending the ENG.md sugar-changes-nothing rule to the third layer of sugar. The ungated micro-convergence test gets a Fit twin; the gated MNIST run gets a Fit arm reaching the M21 numbers.
6. **Validation runs on a throwaway tape via the existing Predict path**, so `TrainingOnly` layers (dropout) are structurally excluded from validation loss exactly as they are from inference — no mode flag appears anywhere.

## Risks / Trade-offs

- [Fit's loop drifts from the documented hand loop and the digit-for-digit gate breaks] → that is the gate working; the fix is in Fit, never in the gate.
- [Config surface invites creep (schedules, callbacks, stopping)] → non-goals list them as recorded follow-ups; the change refuses them in v1 rather than absorbing them silently.
- [Per-epoch mean loss requires accumulating batch losses] → accumulate in float64 on the host; it is reporting, not training state, and never feeds back into parameters.
- [Users pass mismatched tensor shapes] → construction-time validation with errors naming the dimension, consistent with the layer surface's fail-at-construction posture.

## Open Questions

- None blocking. Learning-rate schedules and early stopping are recorded follow-ups, deliberately outside v1.
