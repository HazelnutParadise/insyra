# Change: Reverse-mode autodiff for the MLP family, gradients proved against PyTorch

## Why

M18's verification signal is first-step gradients matching PyTorch under
fixed SafeTensors-loaded weights. The SafeTensors reader landed; this change
is the training core: a tape, VJPs for the MLP operator family, a fused
softmax–cross-entropy loss, and one SGD step, with the PyTorch parity
harness as the verifiable output. Adam and the attention/CNN gradients are
later slices — this change proves the architecture on the smallest family
that exercises every part of it.

The architecture is decided and recorded in ENG.md: a tape of plain
functions, not a graph transform. VJPs are plain functions on tensors, the
tape wrapper is one more caller of the existing kernels, and the inference
path is untouched.

## What Changes

- A tape: training wrappers record (op, inputs, output) as forward runs;
  `Backward` walks the tape in reverse calling each op's VJP, accumulating
  gradients per recorded tensor. Gradients are f32, like everything else.
- VJPs for the MLP family: MatMul (2-D), Add (with broadcast reduction of
  the gradient), Relu, Sigmoid, Tanh, and Gemm's alpha/beta/transpose cases
  actually used by exported MLPs — plus a fused SoftmaxCrossEntropy loss
  (forward returns the mean loss; backward emits the stable softmax−onehot
  gradient), because the separated form loses the cancellation that makes
  it stable.
- SGD: `w -= lr * grad`, applied to the tape's tracked parameters. No
  momentum, no weight decay, no Adam — the first slice proves the loop.
- The parity harness: a Python script (torch is installed in the crosslang
  venv) builds a fixed two-layer MLP, saves its weights via safetensors,
  runs one forward/backward on a fixed batch, and emits every parameter
  gradient plus the loss. The Go test loads the same weights with
  `LoadSafeTensors`, replays the same forward/backward on the tape, and
  compares loss and every gradient within f32 tolerance. Gated through
  `internal/reftest` like every other reference.
- A pure-Go finite-difference check (ungated) on a tiny network, so the
  tape is verified even where the venv is absent.
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No Adam (next slice), no attention/CNN gradients, no training on the
  device (device work is measurement-gated), no minibatch data pipeline,
  no Python-side training-loop comparison beyond the first step.

## Impact

- Affected specs: new `dl-training` capability.
- Affected code: new tape/VJP/optimizer files in `dl` (naming decided by
  implementation, following the package's existing file layout), a new
  Python fixture under `dl/testdata`, docs, changelogs, skills.
