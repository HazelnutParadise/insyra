# Change: The loss and optimizer toolkit

## Why

Training recipes need more than one loss and one schedule. This change
(M28) rounds out the practitioner set, each piece with a torch-verified
contract, none of it new architecture — the tape and optimizer state
plumbing already exist.

## What Changes

- Losses on the tape: `MSELoss(pred, target)` (mean reduction) and
  `BCEWithLogitsLoss(logits, targets)` (the fused stable form — the
  separated sigmoid+log path is not offered, matching the
  softmax–cross-entropy precedent), both with VJPs, finite differences,
  and torch parity.
- `SGDMomentum(lr, momentum)` with torch's velocity convention
  (v = μ·v + g; w -= lr·v), per-parameter state like Adam's.
- `CosineAnnealingLR(initial, tMax)` matching torch's schedule values,
  and `ClipGradNorm(maxNorm)` clipping the global norm across tracked
  parameters before a step, matching torch's total-norm semantics.
- A gated multi-step trajectory: a small net trained with
  BCEWithLogitsLoss + SGDMomentum + CosineAnnealingLR + ClipGradNorm
  matching torch at every step on loss, clipped grad norm, lr, and
  parameters.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No further losses/optimizers (Huber, RMSprop, …) until demand; no
  warmup composition helpers; no per-parameter-group options.

## Impact

- Affected specs: `nn-training`
- Affected code: nn autodiff/optimizer files, a fixture, docs,
  changelogs, skills.
