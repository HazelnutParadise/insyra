## Why

A classifier step ends in `SoftmaxCrossEntropy`. Until it runs on the device, the logits must leave it to compute the loss.

## What Changes

- In `accel.PrecisionFloat32Rounded`, `Tape.SoftmaxCrossEntropy` and its gradient run on device tensors and on the host with identical bits, as a fixed sequence of correctly rounded steps (uses `correctly-rounded-exp-log`).
- An int64 label tensor can be moved to a device (held as int32). Labels are validated once, when moved, and the largest label is recorded on the handle; the tape compares it with the class count, so no kernel needs to report an error.

## Capabilities

### Modified Capabilities
- `nn-training`: defines `SoftmaxCrossEntropy` in `PrecisionFloat32Rounded`.
- `nn-device-tensors`: int64 labels on a device (modifies "a tensor knows where it lives": `To` accepts int64).

## Impact

- Code: `nn/autodiff.go`, `nn/device_tensor.go`, `accel/` (int32 buffers, kernels). API: additions only. Docs and changelogs.
