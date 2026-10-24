# Design: accel-custom-kernels

## Context

`accel/internal/wgpu` already runs every resident kernel through one `kernel` type: a WGSL body, `nRead` read-only storage buffers, `nWrite` read-write ones and a 64-byte uniform of sixteen `u32`. Each call to `run` encodes one dispatch and submits it. `nn` holds resident tensors as `accel.DeviceBuffer`, and `Tape.Custom` records a caller's operation and VJP, but only on host tensors. CoImNet's batched core (`backend/webgpu` in that repository) is seven kernels of at most five read and three read-write buffers each, plus a 64-byte uniform, dispatched 28 steps forward and backward per minibatch. It uses the same `gogpu/wgpu` v0.34.5 that Insyra uses.

## Goals / Non-Goals

**Goals**
- A caller's kernel reads and writes Insyra's resident buffers in the same submission stream as Insyra's own kernels, so a training step never leaves the device.
- A caller's operation sits on a tape with a gradient, and runs on the host when there is no device.
- The precision of every step stays stated and checkable.

**Non-Goals**
- Exporting `gogpu/wgpu` types or the device handle. That would put a third-party version into Insyra's public API and let a caller bypass the memory budget and the submit lock.
- A WGSL compiler, validator or optimiser of Insyra's own. Shader errors come back as `ErrShaderCompile` with the backend's message.
- Making a caller's raw-`f32` kernel bit-identical across devices. Insyra cannot. It states which operations carry that claim.

## Decisions

**1. Reuse the internal kernel layout as the public one.** `KernelSpec{Name, WGSL string; Reads, Writes int; Library bool}`. Bindings are `0..Reads-1` read-only storage, then `Writes` read-write storage, then the uniform at binding `Reads+Writes`. That is exactly `kernel.build`, so the public type wraps the internal one and nothing new is invented. `NewKernel` refuses `Reads+Writes` above the adapter's `maxStorageBuffersPerShaderStage` (8 on the WebGPU default), an empty name or empty source. Compilation happens on first `Run`, once per process, like the built-ins.

**2. One submission for many dispatches.** `accel.Run(dispatches ...Dispatch)` takes the submit lock once, builds every bind group and uniform, encodes all dispatches into one command encoder (one compute pass each, in order), and submits once. WebGPU orders passes within a submission, so dispatch *i* sees what *i−1* wrote. Insyra's own `run` becomes `Run` of one dispatch, so there is one code path. The pending-object bound (`maxPendingObjects`) applies per call: a call that would exceed it synchronises first, as `run` does now. Errors are returned before anything is submitted when a buffer is released, on another device, or of the wrong count.

Alternative considered: one submission per dispatch, as today. CoImNet's 28-step loop would then cost about 300 submissions per minibatch. CoImNet's own backend chose one submission per episode for that reason.

**3. Bit-pattern buffers.** `NewDeviceBufferBits(device, []uint32)` and `BitsToHost() ([]uint32, error)` sit beside the float32 constructor and `ToHost`. The device buffer is the same type. Only the host conversion differs, so a kernel may bind either as `array<u32>`, `array<f32>` or `array<i32>`. Budget accounting is by bytes, as now.

**4. `DeviceOp` has both halves, always.**
```go
type DeviceOp struct {
    Name      string
    Precision accel.Precision // PrecisionFloat32Rounded or PrecisionFloat32
    CPU       func(inputs []*Tensor) (*Tensor, VJP, error)
    Device    func(inputs []*Tensor) (*Tensor, VJP, error)
}
type VJP func(upstream *Tensor) ([]*Tensor, error)
func (t *Tape) Apply(op DeviceOp, inputs ...*Tensor) (*Tensor, error)
```
`Apply` checks the inputs' location the same way `Tape.MatMul` does: all on one device, or all on the host, otherwise an error naming both. It calls the matching half, records the operation with its VJP, and takes ownership of the output so `Clear` and `Close` release it. A `DeviceOp` without `CPU` is refused at `Apply`, so a program that works with a device also works without one. The VJP of the device half must return resident gradients, and the CPU half host ones. `Apply` checks this, so a gradient never silently changes place.

**5. Precision is declared, then enforced or labelled.**
- A tape in `PrecisionFloat32Rounded` accepts an operation that declares `PrecisionFloat32Rounded`. The claim that both halves agree bit for bit is the caller's. `nn.CheckDeviceOp(op, inputs, upstream)` runs both halves and their VJPs and returns the first differing element, so the claim is a test the caller writes. A kernel built with `Library: true` can use the same exact register (`exact_add_product`, `exact_round`, the IEEE helpers) that Insyra's own kernels and their CPU twins use, which is how the claim can be made true. ENG.md's rule ("a kernel that needs more exact arithmetic extends this library") is extended to callers.
- An operation that declares `PrecisionFloat32` is refused by a `PrecisionFloat32Rounded` tape unless `Tape.AllowDeviceDependent()` was called. Once one has run, `Tape.DeviceDependent()` reports true and the accel report records the operation's name. A caller and a reader can then see that this step's numbers depend on the device.

**6. Lending and adopting buffers.** `Tensor.DeviceBuffer()` returns the resident buffer of a float32 tensor without copying. It is an error for a host tensor, or for an int64 tensor (held as int32). The tensor keeps ownership: releasing the tensor releases the buffer. `NewDeviceTensor(buffer, shape)` adopts a buffer, checking that the shape matches its length. The tensor then owns it.

## Risks / Trade-offs

- **A caller's kernel can corrupt Insyra's buffers.** A read-write binding can write any element. This is the price of sharing memory, and is why the API takes `DeviceBuffer`s Insyra allocated and accounts for, never raw handles. Out-of-bounds writes are clamped by WebGPU's robust buffer access, not by Insyra.
- **The Rounded claim of a custom op is only as good as the caller's test.** Mitigated by `CheckDeviceOp` and by labelling every `PrecisionFloat32` step. Insyra's own operations keep their exhaustive verification.
- **Binding limits.** Eight storage buffers per stage is the WebGPU default. CoImNet's widest kernel uses eight, so it fits. A caller needing more splits a kernel.
- **Pending-object growth.** Every dispatch creates a uniform and a bind group. A 300-dispatch `Run` stays under the existing bound, and a larger one synchronises first.

## Verification

- Host and stub tests for every refusal: counts, released buffers, mixed devices, missing CPU half, a `PrecisionFloat32` op on a Rounded tape, a VJP that returns gradients on the wrong side.
- Hardware-gated (`INSYRA_ACCEL_GPU_TESTS=1`) on the M3:
  - A two-kernel `Run` whose second dispatch reads the first's output.
  - A `Library: true` kernel computing an exact dot product, compared bit for bit with Insyra's `RoundedMatMul`.
  - A `DeviceOp` whose halves are that kernel and its CPU twin, through `CheckDeviceOp` and a full tape `Backward`.
- A no-device run of the same `DeviceOp` returns the same bits through the CPU half.
- `go test ./...` in the core module and in `accel/backend/wgpu`.

## Migration Plan

Additive. Nothing existing changes signature or behaviour. CoImNet adopts it behind its own opt-in flag.

## Open Questions

- Whether `Run` should also accept Insyra's built-in operations as dispatches, so a caller can interleave them without a synchronisation. Not needed for CoImNet's first use: the tape orders Insyra's operations and the caller's `Run` in one queue already, because both go through the same submit lock and queue.
