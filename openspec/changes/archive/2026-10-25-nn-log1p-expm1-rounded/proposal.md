# Proposal: nn-log1p-expm1-rounded

## Why

The soft-target and distillation losses in `PrecisionFloat32Rounded` need `log1p` and `expm1` that are correctly rounded on the host and on a device, the way `exp` and `log` already are. Barron's second fix (arXiv 2610.31642) is written in `expm1` and `log1p`, and so is a cross-entropy whose loss does not read 0 once the largest probability rounds to 1.

## What Changes

- Correctly rounded float32 `log1p` and `expm1` on the host (a float64 evaluation that settles the rounding, with a 416-bit fallback) and as WGSL integer kernels on the device.
- `accel.RoundedLog1p` and `accel.RoundedExpm1` on resident buffers.
- Not in this change: the losses that use them.

## Capabilities

### Modified Capabilities

- `nn-correctly-rounded-functions`: two more functions.

## Impact

`nn/exp_log_rounded.go`, `accel/internal/wgpu/fixed_math.go`, `resident_kernels.go`, `accel/resident.go`, tests, both CHANGELOGs.
