## Why

M38 of #379, one operation: a recurrent step `tanh(W·state + input)` on resident tensors could run `MatMul`, `Add` and `EdgeSum` on the device but had to stop at `Tape.Tanh`, which refused a device tensor. M37 added the device `tanh`; the tape can now use it.

## What Changes

- `Tape.Tanh` on a resident float32 tensor, in `PrecisionFloat32Rounded` (or `PrecisionFloat32`), runs `accel.RoundedTanh` and keeps its output and gradient on the device.
- `accel.RoundedTanhGrad(y, upstream)` computes `RN(upstream · RN(1 − RN(y·y)))` on the device, each step an exact product or sum rounded once.
- In `PrecisionFloat32Rounded` a zero at any step of the tanh gradient is `+0`, the rule every other operation of that mode follows, so the host and the device agree. **Breaking** in that mode only: a gradient that was `-0` (a negative upstream where `tanh` rounds to `±1`) is now `+0`. The default tape keeps IEEE's `-0`.

## Impact

- `nn/autodiff.go`, `nn/autodiff_device.go`, `accel/resident.go`, `accel/internal/wgpu/resident_kernels.go`, `accel/internal/wgpu/warmup.go`, their tests; `Docs/nn.md`, `Docs/accel.md`, both changelogs, `delivery-status.md`. API: one addition, `accel.RoundedTanhGrad`.
