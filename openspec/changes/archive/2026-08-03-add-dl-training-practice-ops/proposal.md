# Change: Training practice ops — dropout, decoupled weight decay, an LR schedule

## Why

The loop converges, but real training practice regularizes and
schedules. Three pieces close the gap between "converges" and "trains
the way practitioners train", each with a PyTorch-verified contract
(M22).

## What Changes

- **Dropout** as a tape operation: training mode masks with probability
  p and scales by 1/(1−p) (inverted dropout, PyTorch semantics), from
  the tape's seeded RNG so runs stay deterministic; the VJP routes
  gradients through the same mask. Inference/eval is the identity —
  and the ONNX graph runner is untouched (Dropout in inference graphs
  is already identity per the spec; verify the operator table's claim
  and add the graph-level identity only if a real graph carries it).
- **Decoupled weight decay** (AdamW): `w -= lr·λ·w` applied separately
  from the Adam moment update, matching `torch.optim.AdamW` — not the
  coupled L2-in-gradient form, and the difference is asserted in the
  parity fixture.
- **An LR schedule**: step decay (factor γ every k steps) as a small
  helper the optimizer reads each step, verified against
  `torch.optim.lr_scheduler.StepLR` over several steps. One schedule
  proves the seam; more schedules wait for need.
- The proof, both established ways: ungated tests (mask statistics and
  scaling, determinism under seed, decay arithmetic, schedule values)
  and a gated PyTorch fixture training a small MLP several steps with
  dropout (fixed mask via a shared seed is not portable — instead the
  fixture DISABLES dropout for the numeric comparison and verifies
  AdamW+StepLR trajectories exactly, while dropout parity is asserted
  on mask statistics and eval-mode identity, the standard practice).
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No other schedulers (cosine, warmup) or optimizers, no BatchNorm
  training mode, no data-augmentation pipeline.

## Impact

- Affected specs: `dl-training`
- Affected code: dl autodiff files, a new Python fixture, docs,
  changelogs, skills.
