## Why

After `device-resident-tape-step`, `SGD` runs on the device but `Adam` and `AdamW` do not, so a step that uses them must leave the device for the optimizer. The default tape's AdamW uses float64 intermediates a GPU does not have.

## What Changes

- In `accel.PrecisionFloat32Rounded`, `Tape.Adam` and `Tape.AdamW` run on device tensors and on the host with identical bits, with moment state on the device. Each is a fixed sequence of float32 steps, every step the correctly rounded value of its exact operation (division and square root need no new approximation: IEEE division is correctly rounded, and `float32(math.Sqrt(float64(x)))` is provably correctly rounded for a float32 `x`).
- The scalars (`1-beta1`, `1-beta2`, the bias corrections, `lr*wd`) are computed once on the host and rounded once to float32, so host and device read the same values.
- Device kernels for division and square root, in integer arithmetic.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `nn-training`: defines `Adam` and `AdamW` in `PrecisionFloat32Rounded`.
- `nn-device-tensors`: Adam and AdamW accept device tensors.

## Impact

- Code: `nn/autodiff.go` and `nn/autodiff_device.go` (host path and dispatch), `accel/internal/wgpu/` (division, square root, Adam kernels), `accel/` (public function).
- API: additions only; the default tape's AdamW is unchanged.
- Docs: `Docs/nn.md`, both changelogs, `delivery-status.md`.
