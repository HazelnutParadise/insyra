# Proposal: nn-softmax-gradient-past-the-cliff

## Why

The default-precision fix of the label's cross-entropy gradient (Barron, arXiv 2610.31642) left two places with the same defect, and the owner ruled on 2026-10-25 that both are fixed: `PrecisionFloat32Rounded` cross-entropy, on the host and on the device, and the standalone `Softmax` backward.

## What Changes

- In `PrecisionFloat32Rounded`, the label's gradient is `-(o/s)` times `upstream/rows`, with `o` the exact sum of the other exponentials rounded once and `s` the exact sum of all of them rounded once, each step one correctly rounded operation. The host and the device compute the same bits.
- The standalone `Softmax` backward uses the largest probability as the pivot: `u_peak - sum(y*u)` is computed as the sum over the other classes of `y_j*(u_peak - u_j)`, so the peak's gradient no longer rounds to 0.
- Not in this change: the rounded-mode loss, which needs a correctly rounded `log1p` that does not exist yet; and soft-target cross-entropy (the paper's second bug), a new API with its own change.
- **Behaviour change**: the label's gradient in `PrecisionFloat32Rounded` has different bits for every input, not only past the cliff.

## Capabilities

### Modified Capabilities

- `nn-training`: the standalone `Softmax` backward keeps its peak gradient.
- `nn-device-tensors`: rounded cross-entropy keeps the label's gradient, equal on host and device.

## Impact

`nn/softmax_ce_rounded.go`, `accel/internal/wgpu/softmax_ce.go`, `nn/autodiff_attention.go`, tests, `Docs/nn.md`, both CHANGELOGs.
