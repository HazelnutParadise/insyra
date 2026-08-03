# Change: The complete layer catalog — CNN, normalization, embedding

## Why

Sequential covers the MLP family; completeness (M26) means every
architecture the tape can train composes as layers. Two pieces are
genuinely new math, the rest are wrappers:

- **Training-mode BatchNorm2D.** The CNN-gradients change refused
  batch-statistics gradients — a scope line, not a law. A complete CNN
  training surface needs them: the full three-term gradient through
  batch mean and variance, plus running-statistics updates with
  momentum, verified against `torch.nn.BatchNorm2d` in train mode,
  with eval mode using the running statistics exactly as the existing
  inference kernel does.
- **Embedding.** Token lookup with a scatter-add gradient into the
  table, verified against `torch.nn.Embedding`.

## What Changes

- New layers, torch-named, torch-layout at the load boundary:
  `Conv2D(in, out, kernel, opts)` (pads/strides/dilations/groups,
  bias; weight layout `[out,in,kh,kw]` matches torch — no transpose),
  `MaxPool2D`, `AvgPool2D` (count_include_pad), `GlobalAvgPool`,
  `BatchNorm2D(features)` (train forward uses batch statistics and
  updates running stats; Predict uses running stats), `LayerNorm`,
  `Embedding(vocab, dim)`.
- New tape ops with VJPs: training-mode BatchNorm (three-term
  gradient; running-stats update as a documented side effect) and
  Embedding lookup (scatter-add). Both pinned by ungated finite
  differences and gated PyTorch parity.
- The refusal of training-mode batch statistics in the raw kernel path
  is superseded by the new tape op; the inference kernel and the ONNX
  graph runner are untouched.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No RNN/attention layers (the encoder composes via Func today; a
  dedicated attention layer waits for demand), no Conv1D/3D, no
  quantization-aware anything.

## Impact

- Affected specs: `nn-training`
- Affected code: nn layer and autodiff files, fixtures, docs,
  changelogs, skills.
