## Why

A training step on a weak host spends its time moving data and running operations that have no device kernel (issue 494, "keep Tape tensors resident on the accelerator"). Today only `MatMul` has a device path, and each call uploads both operands and reads the result back. This is milestone M38 ("Tensors stay on the device") of `delivery-status.md`, widened from `EdgeSum` to the operations a training step needs. The owner agreed the design on 2026-10-20 (comment on the issue).

## What Changes

- A `*nn.Tensor` can live on a device: `Tensor.To(dev)` uploads, `Tensor.ToHost()` reads back, `Tensor.Device()` reports where it lives (the zero value is the host).
- Tape operations dispatch on where their operands live. This change covers the first group: `MatMul` (transposed operands handled in the kernel, so no VJP materialises a `Transpose`), `Add` with broadcast and its broadcast-reduce gradient, gradient accumulation, and `SGD`.
- `Tape.Clear()` drops the recorded operations and step intermediates and keeps parameters and optimizer state; `Tape.Close()` releases everything the tape holds on a device.
- `Tape.SetPrecision` / `GetPrecision` select the precision mode. The new `accel.PrecisionFloat32Rounded` defines every result as the exact value rounded once to float32, so CPU, GPU and the CPU fallback return the same bits. The default tape is unchanged.
- Host views of a resident tensor (`Float32Data`, `Parameter.Update`, `Parameter.Grad`) return an error that names `ToHost`. An operation with no device kernel downloads, runs on the host, uploads and records a fallback reason; strict GPU mode refuses.
- Resident tensors are charged to the accel memory budget.
- `SoftmaxCrossEntropy`, `AdamW` (defined entirely in float32 in this mode), `Embedding` and the remaining operations are later changes, one verifiable result each.

## Capabilities

### New Capabilities
- `nn-device-tensors`: tensors that live on a device, explicit movement, tape dispatch by residency, lifetime (`Clear`, `Close`), reporting and strict mode.

### Modified Capabilities
- `accel-exact-arithmetic`: adds `PrecisionFloat32Rounded` and the rule that a tape in that mode returns the same bits on every device and on the CPU fallback.
- `nn-training`: adds `Tape.Clear`, `Tape.Close`, `Tape.SetPrecision` and `Tape.GetPrecision`.

## Impact

- Code: `nn/tensor.go`, `nn/autodiff.go`, `nn/device_matmul*.go`, new `nn/device_tensor*.go`; `accel/types.go`, `accel/nn_*.go`, `accel/internal/wgpu/*` (resident buffers, MatMul on buffers, elementwise and reduce kernels).
- API: additions only; no released result changes. `PrecisionFloat32Rounded` is opt-in.
- Docs: `Docs/nn.md` (both modes, when to use which), `Docs/Configuration.md` if a setting changes, `Docs/accel.md`; `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`, `README` only if a package row changes.
- Known limit carried over: device MatMul's bit-parity rests on behaviour WGSL does not promise (`AGENTS.md` follow-up 2026-09-25). This change keeps MatMul as it is and measures parity where it runs; moving MatMul to the exact rule is a separate decision.
