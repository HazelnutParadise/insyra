# Proposal: accel-custom-kernels

## Why

A model whose own core runs on the GPU cannot keep a whole training step on one device today. CoImNet trains a recurrent core in its own WebGPU kernels and uses Insyra's `nn` tape for the encoder, readout, loss and AdamW. The two hold separate GPU contexts, so every step copies tensors device to host to device twice. A profile of one CoImNet epoch on an M3 put the GPU core at about 8% of the time and host matrix work at about 29%, the rest being staging and waiting around those copies (issue #546).

Insyra already holds tensors on the device and runs its own kernels on them. What is missing is a way for a caller to run its kernel on those same buffers, and to put that kernel on a tape with a gradient.

## What Changes

- **`accel` runs caller-supplied kernels.** `accel.NewKernel(KernelConfig{Name, WGSL, ReadOnly, ReadWrite})` holds a WGSL compute shader with the binding layout Insyra's own resident kernels use: `ReadOnly` read-only storage buffers, then `ReadWrite` read-write ones, then one 64-byte uniform of sixteen `u32`. `accel.Submit(dispatches ...Dispatch)` encodes the dispatches in order into one submission on the resident device, under the lock and queue Insyra's own kernels share.
- **`DeviceBuffer` carries bit patterns and can be rewritten.** `accel.NewBitsBuffer` and `DeviceBuffer.BitsToHost` move indices without a float conversion; `Write`, `WriteBits` and `Zero` overwrite a buffer in place after waiting for earlier kernels.
- **`accel.ResidentLimits()`** reports the device's storage-buffer count, buffer size, workgroup count and invocation limits.
- **`Tape.Custom` accepts resident tensors** on a tape in the new `accel.PrecisionFloat32` mode, which `SetPrecision` now accepts. Built-in operations compute in that mode exactly as in `PrecisionFloat32Rounded`; the mode only adds permission to record an operation whose kernel Insyra cannot check, and says plainly that the step's bits depend on the device. The tape owns a resident output and the gradients the reverse rule returns.
- **`Tensor.DeviceBuffer()` and `nn.NewDeviceTensor(shape, buffer)`** lend a resident tensor's buffer to a kernel and adopt a buffer a kernel wrote.
- Not in this change, on purpose: a required host implementation per operation, a self-declared `PrecisionFloat32Rounded` for caller kernels, and the exact-arithmetic WGSL library for callers. A review (2026-10-24) found the first unenforceable and the second a promise nothing checks; the library can be added when a caller brings an exact kernel. Reusing bind groups across calls was also left out until measured, because rewriting a uniform that an in-flight dispatch still reads is unsafe on shared-memory GPUs without a proof per backend.
- Nothing about existing operations, their kernels or their results changes.

## Capabilities

### New Capabilities

- `accel-custom-kernels`: caller-supplied WGSL kernels on resident buffers, batched into one submission, with bit-pattern buffers, in-place writes and device limits.

### Modified Capabilities

- `nn-device-tensors`: `Tape.Custom` records resident tensors in `PrecisionFloat32`; tensors lend and adopt device buffers.

## Impact

- New: `accel/custom_kernel.go`, `accel/internal/wgpu/custom.go`, tests including hardware-gated ones. `accel/internal/wgpu/resident.go` routes its single-dispatch path through the shared submit.
- `nn`: `Custom`, `SetPrecision`, `DeviceBuffer`, `NewDeviceTensor`.
- `Docs/accel.md`, `Docs/nn.md`, ENG.md and both CHANGELOGs.
- No breaking change for existing calls. A `Custom` call on a closed tape now returns an error, as the other tape operations do.
