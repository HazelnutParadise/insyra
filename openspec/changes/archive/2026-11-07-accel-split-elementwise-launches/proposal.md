## Why

M38's recurrent-step measurement on a GT 1030 lost the device at a million nodes with a batch of 16: the elementwise resident kernels run a whole array in one dispatch, and Windows resets a GPU whose dispatch runs about two seconds. On the M3, `tanh` over 2^24 elements took 2.85 s in one launch and `exp` 1.40 s. `EdgeSum` was already split this way in M36.

## What Changes

- The elementwise kernels in `resident_kernels.go` (add, the tanh gradient, SGD, divide, square root, Adam, and the fixed-point exp, log, expm1, log1p and tanh) read a `base` index and are launched through `runElementwise`, each launch its own submission: at most 2^18 elements for the fixed-point functions and 2^21 for the exact-register ones. Results are unchanged.
- `TestRecurrentStepTiming` times a training step of `tanh(EdgeSum(W, s) + input)` unrolled four times, on the default tape, the rounded tape on the host and the rounded tape on the device.
- The cross-entropy kernels are not split; that is recorded as a follow-up.

## Impact

- `accel/internal/wgpu/resident_kernels.go` and a test; `nn/recurrent_step_timing_test.go`; both changelogs, `AGENTS.md` (follow-up), `delivery-status.md`. API: none.
