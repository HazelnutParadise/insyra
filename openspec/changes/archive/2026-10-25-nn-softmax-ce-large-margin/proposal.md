# Proposal: nn-softmax-ce-large-margin

## Why

Barron, "A Fix for Softmax and Cross-Entropy" (Google DeepMind, arXiv 2610.31642, 2026-10-23) shows that computing a hard-target cross-entropy gradient as p - 1 rounds the label's gradient to exactly 0 once the label's margin passes the precision cliff (about 16.6 in float32), while the other classes' gradients stay nonzero and the row stops summing to zero. `Tape.SoftmaxCrossEntropy` on the host did exactly that: with logits `[20, 0]` the label's gradient was 0 instead of -2.06e-9.

## What Changes

- The host `SoftmaxCrossEntropy` backward takes the label's gradient as -(sum of the other exponentials)/(sum of all), the form the paper gives, so it stays accurate past the cliff and the row sums to zero.
- The forward loss is softplus(-m) with m = z_label - log(sum of the other exponentials), so it no longer rounds to 0 past a margin of about 36.
- Not in this change: `PrecisionFloat32Rounded` (host and device), which has the same defect and needs its own design because its results are specified bit for bit; the standalone `Softmax` backward.

## Capabilities

### Modified Capabilities

- `nn-training`: the fused cross-entropy keeps its gradient at large margins.

## Impact

`nn/autodiff.go`, `nn/softmax_collapse_test.go`, both CHANGELOGs. No signature changes. No pinned convergence number in the `nn` tests moved.
