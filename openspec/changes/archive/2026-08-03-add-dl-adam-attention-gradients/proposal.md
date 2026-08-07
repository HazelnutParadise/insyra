# Change: An encoder block trains one Adam step, and PyTorch agrees

## Why

The tape holds for the MLP family. M18's remaining distance to "training"
is the attention family's gradients and a real optimizer. This change closes
both with one verifiable output: a fixed two-head transformer encoder block —
the same architecture the inference proof runs — takes one Adam training
step in dl, and its loss, every parameter gradient, and every post-step
parameter value match PyTorch under identical SafeTensors-loaded weights.
That statement pins every new VJP and the optimizer in a single measurable
claim, the way the encoder parity proof pinned the attention kernels.

## What Changes

- VJPs for the attention family, as plain functions like the existing ones:
  batched MatMul (batch-shape broadcasting in the backward, gradients
  reduced over broadcast batch dims), axis-Softmax, LayerNormalization
  (input, scale, and bias gradients), Gelu (exact erf form; tanh form only
  if the fixture uses it), Erf, Sqrt, Pow, ReduceMean, and the shape ops the
  encoder needs to backpropagate through — Transpose (permutation inverse),
  Reshape/Flatten (shape restore), Squeeze/Unsqueeze, Slice, Concat, Split
  as used by the encoder wrappers. Ops not needed by the encoder proof are
  out of scope; refuse clearly rather than half-implement.
- Adam: bias-corrected first and second moments, per-parameter state,
  matching PyTorch's `torch.optim.Adam` defaults (lr, betas, eps as the
  fixture sets them; no weight decay, no amsgrad).
- The proof, both ways as established: central finite differences ungated
  for every new VJP on tiny shapes; and the encoder one-step parity gated
  through `internal/reftest` — the fixture builds the encoder in torch with
  deterministic weights, saves them via safetensors, runs one forward,
  backward, and Adam step on a fixed batch, and emits loss, gradients, and
  post-step parameters; dl replays from the same file and compares within
  f32 tolerance.
- Inference untouched, as before, verified by the unchanged dl suites.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No CNN gradients (next slice), no training on the device, no dropout, no
  weight decay or amsgrad, no learning-rate schedules, no multi-step
  training-curve comparison.

## Impact

- Affected specs: `dl-training`
- Affected code: dl autodiff files (new VJPs, Adam), a new Python fixture,
  docs, changelogs, skills.
