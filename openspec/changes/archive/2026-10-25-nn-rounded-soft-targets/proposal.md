# Proposal: nn-rounded-soft-targets

## Why

`SoftTargetCrossEntropy` and `DistillationLoss` (the nn-soft-target-cross-entropy change) run only on host tensors in default arithmetic, and the rounded hard-label loss still reads 0 once the label leads by about 16.6, because `log(s)` sees `s = 1 + o` rounded first. With correctly rounded `log1p` and `expm1` (nn-log1p-expm1-rounded) the owner ruled on 2026-10-25 that the GPU versions are made too, including the paper's own distillation formula (Barron, arXiv 2610.31642, Eq. 9).

## What Changes

- In `PrecisionFloat32Rounded`, `SoftTargetCrossEntropy` and `DistillationLoss` compute as fixed sequences of correctly rounded operations and exact sums on host tensors, and run on the device for resident tensors with the same bits.
- The loss uses `ell = log1p(o)`, `o` the exact sum of the exponentials but the first peak's, so a label far past the cliff has a loss that is not 0. This also applies to the hard-label `SoftmaxCrossEntropy` loss in this mode, whose bits change.
- The distillation gradient is Eq. 9 with each step rounded once: differences formed as one exact four-term sum, `expm1` and `log1p` correctly rounded; a row falls back to `q - p` when a teacher probability is 0 or a centred difference exceeds 1.
- A resident targets tensor is not read back, so a row that is not a distribution is not refused there.
- **Behaviour change**: the rounded hard-label loss has different bits whenever `log(s)` and `log1p(o)` differ.

## Capabilities

### Modified Capabilities

- `nn-device-tensors`: rounded cross-entropy loss, soft-target and distillation losses.

## Impact

`nn/soft_ce_rounded.go`, `nn/softmax_ce_rounded.go`, `nn/autodiff_soft_ce.go`, `nn/autodiff_device.go`, `accel/internal/wgpu/soft_ce.go`, `softmax_ce.go`, `accel/resident.go`, tests, `Docs/nn.md`, both CHANGELOGs.
