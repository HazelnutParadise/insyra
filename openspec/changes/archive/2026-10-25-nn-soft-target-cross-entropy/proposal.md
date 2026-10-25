# Proposal: nn-soft-target-cross-entropy

## Why

`Tape.SoftmaxCrossEntropy` takes only hard int64 labels. Distillation, label smoothing and mixup need a cross-entropy against a probability row, and today a user has to build `-sum(p * log(softmax(z)))` from `Softmax`, `Log` and `Mul`, whose backward loses the gradient exactly where Barron ("A Fix for Softmax and Cross-Entropy", arXiv 2610.31642) shows it does: the second bug, Softmax Cancellation. When the student's logits are close to the teacher's, `q - p` subtracts two nearly equal probabilities; and when the forward pass builds `p` by one sequence of operations and the backward by another, equal logits still leave a nonzero gradient.

## What Changes

Two new tape methods, both mean over rows like `SoftmaxCrossEntropy`, both float32 host tensors only.

```go
// Label smoothing, mixup: targets is a probability row per sample.
targets, _ := nn.NewTensor([]int{n, classes}, smoothed) // each row sums to 1
loss, err := tape.SoftTargetCrossEntropy(logits, targets)

// Distillation: the teacher is a constant, given as logits so that
// student - teacher is exact. Temperature is the caller's division of both.
loss, err := tape.DistillationLoss(studentLogits, teacherLogits)
```

- `SoftTargetCrossEntropy(logits, targets)`: loss = mean over rows of `-sum_k p_k * logsoftmax(z)_k`; gradient `(q - p)/rows` times upstream. Rows of `targets` must be finite, non-negative and sum to 1 within 1e-5; otherwise an error naming the row and the reason. A smoothed or mixed row is only as exact as its float32 values, so near-equality accuracy is bounded by that row. Gradient for `targets` is nil (a constant).
- `DistillationLoss(student, teacher)`: both float32 `[N,C]`. `p = softmax(teacher)` and `q = softmax(student)` come from one shared float64 function that uses `max + log1p(sum of the others)`; the loss is `-sum p*logsoftmax(student)` (skipping `p_k = 0`, so `0*log 0 = 0`), and the gradient is `q - p` with the largest logit's entry taken as minus the sum of the others (the paper's fix for the first bug, applied here too). Equal logits give exactly 0. The teacher receives no gradient. `+Inf` or `NaN` in a row gives NaN, as the hard-label path does.
- The paper's Eq. 9 (`expm1`/`log1p` on `delta = z_S - z_T`) computes the distillation gradient when every teacher probability is nonzero and every centred difference is at most 1, where plain `q - p` subtracts two nearly equal probabilities and can lose a difference float32 could hold (a student `[1e-20, 0]` against a teacher `[0, 0]` has gradient 2.5e-21, not 0; Codex found this on review). Outside that range, for example a teacher `[1000, 0, 0]` whose probabilities underflow to 0 where Eq. 9 gives NaN, `q - p` from one shared function is accurate and is used. Equal logits give exactly 0 on both paths.
- Under `PrecisionFloat32Rounded` both keep the default arithmetic on host tensors, as `MSELoss` and `BCEWithLogitsLoss` do; resident tensors are refused with the usual error.
- `nn.Fit` is unchanged. A `Loss` selector for these is not in this change.

## Decisions to review

1. Two methods or one? One `SoftTargetCrossEntropy` cannot give the paper's fix, because the fix needs `z_T`, not `p`. Two keeps each honest about what it can promise.
2. Names: `SoftTargetCrossEntropy` follows the `SoftmaxCrossEntropy` family; `DistillationLoss` names the use. `KLDistillation` was rejected: the gradient of cross-entropy and of KL against a constant teacher are the same, but the loss values differ by the teacher's entropy, and the paper's loss is the cross-entropy.
3. Temperature stays with the caller (divide both logits). A `Temperature` argument would be a second place to get the `T^2` gradient scaling wrong.
4. Float64 inside, float32 out, as the hard-label path does.
5. Tests: a float64 PyTorch reference for both losses and gradients, equal logits give exactly 0, near-equal logits, `sum(g) = 0` per row, a large margin, teacher `[1000, 0, 0]`, a zero-probability target with a `-Inf` logit, one class.

## Capabilities

### New Capabilities

- `nn-soft-target-cross-entropy`: cross-entropy against a probability row and against a teacher's logits.

## Impact

`nn/autodiff.go` (or a new `nn/autodiff_soft_ce.go`), tests (a PyTorch reference for both losses and gradients, the equal-logits case, the near-equal case, and the large-margin case), `Docs/nn.md`, both CHANGELOGs. Additive.
