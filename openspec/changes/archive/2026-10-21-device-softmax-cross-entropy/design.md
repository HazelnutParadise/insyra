## Context

Depends on `correctly-rounded-exp-log`. The host loss today uses float64 `math.Exp` and `math.Log` (`nn/autodiff.go`); this mode replaces it with the pinned sequence.

## Goals / Non-Goals

**Goals:** classifier loss and gradient on the device, bit-identical to the host. **Non-Goals:** other losses; label smoothing.

## Decisions

- Every step is one correctly rounded operation or an exact sum rounded once; reductions (max, sum) are order-independent.
- Labels validated at `To`, bound checked on the host; a kernel never reports an error.
- Non-finite logits make the row NaN on both sides; the host max skips NaN, so the subtraction yields NaN either way.

## Risks / Trade-offs

- Integer exp and log make the loss costlier than a float kernel; measured and recorded.
