## Why

M37 of #379: `nn.Tanh` is correctly rounded on the host (M34), but the device has no `tanh`, so a recurrent step whose state lives on a GPU cannot apply its activation there. The device already evaluates `exp`, `log`, `expm1` and `log1p` correctly rounded in 128-bit fixed point; `tanh` is one division away from `exp`.

## What Changes

- A WGSL `tanh_rounded` in the fixed-point library: for `2^-13 <= |x| < 9.5`, `e = exp(-2|x|)` from the existing reduction, then `(1 - e) / (1 + e)` by restoring division to 112 fractional bits, rounded once; the ranges outside it return what the host proves (`x`, or `±1`), and a NaN is returned unchanged.
- `accel.RoundedTanh` computes it on a resident array, like `RoundedExpm1`.
- A gated test compares it with `nn.Tanh` on all 2^32 float32 inputs; a sample is checked against a `math/big` oracle in the `wgpu` package.

Wiring `Tape.Tanh` to resident tensors is M38's, operation by operation, and is not in this change.

## Impact

- `accel/internal/wgpu/fixed_math.go`, `accel/internal/wgpu/resident_kernels.go`, `accel/resident.go`, their tests, `nn/exp_log_device_test.go`; `Docs/accel.md`, both changelogs, `delivery-status.md`. API: one addition.
