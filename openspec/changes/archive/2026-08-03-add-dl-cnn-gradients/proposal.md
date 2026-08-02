# Change: CNN gradients — an MNIST-class CNN trains one step, and PyTorch agrees

## Why

This is M18's last slice. The tape differentiates the MLP and attention
families; the CNN family — Conv, pooling, BatchNormalization — is the
remaining gap between "dl trains encoders" and "dl trains the model classes
its inference already runs". One verifiable output closes it, the same shape
as the encoder proof: a fixed MNIST-class CNN takes one Adam step in dl and
PyTorch agrees on the loss, every parameter gradient, and every post-step
parameter under identical SafeTensors-loaded weights.

## What Changes

- Conv VJP (2-D NCHW): input gradient (the transposed correlation of the
  upstream with the weights), weight gradient (correlation of the input with
  the upstream), and bias gradient (sum over batch and spatial dims), under
  the attribute combinations the fixture exercises — explicit pads, strides,
  and groups at minimum; dilations if the fixture uses them; refuse
  unexercised combinations clearly rather than half-implement. Accumulate in
  float64 like the forward.
- MaxPool VJP: route each upstream value to its argmax input position
  (recompute or record positions; ties resolve to the first maximum, which
  is also what the forward selects). AveragePool and GlobalAveragePool VJPs:
  spread upstream over each window under the same count_include_pad
  semantics as the forward.
- BatchNormalization VJP in inference mode: mean and variance are loaded
  constants, so the backward is the elementwise affine chain — input, scale,
  and bias gradients; no batch-statistics gradient (training-mode batch norm
  is out of scope and refused, as the forward already refuses it).
- Pad (constant) VJP: slice the upstream back to the unpadded shape, if the
  fixture's architecture needs it; otherwise refuse with the op named.
- The proof, both established ways: central finite differences ungated for
  every new VJP including groups and asymmetric pads; and the CNN one-step
  parity gated via `internal/reftest` — the torch fixture (eval-mode
  BatchNorm to match inference semantics) with deterministic weights, one
  forward/backward/Adam step on a fixed batch, compared on loss, gradients,
  and post-step parameters within f32 tolerance.
- Inference untouched; docs, changelogs both languages, skills — same
  change. Closing M18 in `delivery-status.md` happens at acceptance.

## Non-Goals

- No training-mode BatchNormalization (batch statistics and their
  gradients), no dropout, no data pipeline, no device training, no
  multi-step curves.

## Impact

- Affected specs: `dl-training`
- Affected code: dl autodiff files (CNN VJPs), a new Python fixture, docs,
  changelogs, skills.
