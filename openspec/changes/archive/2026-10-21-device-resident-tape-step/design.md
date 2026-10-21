## Context

`nn.Tensor` stores host slices only. The one device path, `RegisterDeviceMatMul`, uploads, runs and reads back per call. `Tape` keeps gradients in a host map, and `Parameter` owns its Adam state. `ENG.md` defines a device float32 result as the exact value rounded once, and `delivery-status.md` M38 already decides that tensors carry their device. The owner agreed the surface on 2026-10-20 (issue 494).

## Goals / Non-Goals

**Goals:**
- A tape step of `MatMul` and `Add` with their reverse pass and an `SGD` update runs on a device with no host copy between operations, equal to the CPU bit for bit in `PrecisionFloat32Rounded`.
- A missing or failing device changes speed only.

**Non-Goals:**
- `SoftmaxCrossEntropy`, `AdamW`, `Embedding`, `EdgeSum`, `Tanh` on the device: one change each, after this one.
- Buffer exchange with `Tape.Custom`: refused for resident inputs until the later change.
- Moving the existing device `MatMul` to the exact rule (open follow-up, `AGENTS.md` 2026-09-25).
- A tolerance-based fast mode.

## Decisions

- **Residency is a property of the tensor.** A private `deviceStorage` field sits next to the host slices; `To`/`ToHost`/`Device` are the only public surface. A separate `DeviceTensor` type would duplicate every `Tape` method.
- **The device handle is opaque to `nn`.** `nn` already imports `accel`; storage is an `accel`-owned buffer handle, so `nn` never touches wgpu.
- **Precision is a per-tape setting.** `SetPrecision(accel.PrecisionFloat32Rounded)` selects the new contract; the default tape keeps today's code paths untouched.
- **Rounded MatMul and Add on the device use the exact register** from `accel/internal/wgpu/exact_sum.go`: products and sums are formed in integer limbs and rounded once. The CPU reference in this mode uses the same rule, so a missing device returns the same bits.
- **Transposed operands are read in place** by the kernel, so the VJP of `MatMul` needs no `Transpose`.
- **Sync only at `ToHost`.** Operations are submitted in queue order without waiting; the wait happens on the readback.
- **Lifetime is explicit.** `Clear` frees step intermediates, `Close` frees the rest; a finalizer is a logged backstop only. Resident bytes are charged to the accel memory budget.

## Risks / Trade-offs

- The exact register MatMul may be slower than the current f32 device MatMul. The change measures both on a ladder against the all-core CPU and records the numbers in `delivery-status.md`; if the exact kernel loses to the CPU at the shapes CoImNet uses, the task stops and the owner decides.
- Verified on Apple Metal only (`AGENTS.md` follow-up 2026-08-01); parity on other platforms is carried as the standing follow-up.
- The tape is not safe for concurrent use today; resident tensors add no new sharing and the documentation says so.
