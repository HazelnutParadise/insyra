# Proposal: accel-custom-kernels

## Why

A model whose own core runs on the GPU cannot keep a whole training step on one device today. CoImNet trains a recurrent core in its own WebGPU kernels and uses Insyra's `nn` tape for the encoder, readout, loss and AdamW. The two live on different GPU contexts, so every step copies the encoder's output to the host and into the core, and the core's readout back out to the host and into the tape. A CPU profile of one CoImNet epoch on an M3 put Insyra's host MatMul and the round trip around it at roughly half the CPU time, while the GPU core itself took about 8%. On the 4-core Linux host the same run is 5× slower than on the M3, for the same reason.

Insyra already holds tensors on the device (`Tensor.To`, `accel.DeviceBuffer`) and runs its own kernels on them. What is missing is a way for a caller to run *its* kernel on those same buffers, and to put that kernel on a tape as one operation with a gradient. The user chose this route on 2026-10-24 over exporting raw device handles and over CoImNet reimplementing the encoder, readout, loss and AdamW itself.

## What Changes

- **`accel` gains caller-supplied kernels.** `accel.NewKernel(KernelSpec{Name, WGSL, Reads, Writes, Library})` holds a WGSL compute shader with a fixed binding layout: `Reads` read-only storage buffers, then `Writes` read-write ones, then one 64-byte uniform of sixteen `u32`. This is the layout Insyra's own resident kernels already use. `accel.Run(dispatches ...Dispatch)` encodes the dispatches in order into one submission on the resident device. Each dispatch names its kernel, its `DeviceBuffer`s, its uniform values and its workgroup counts. `Library: true` prepends Insyra's exact-arithmetic WGSL library, so a kernel can produce correctly rounded results.
- **`DeviceBuffer` carries bit patterns.** `accel.NewDeviceBufferBits(device, []uint32)` and `DeviceBuffer.BitsToHost()` move index and topology arrays without reinterpreting them as floats. A buffer is bytes on the device either way. The existing float32 constructor is unchanged.
- **`nn` gains a device operation with a CPU half.** `nn.DeviceOp{Name, Precision, CPU, Device}` is one tape operation with two complete implementations, each returning its output and its VJP. `Tape.Apply(op, inputs...)` runs `Device` when every input is resident on one device and `CPU` when every input is on the host, and records the VJP like `Tape.Custom` does. With no usable device, the caller's tensors stay on the host and the CPU half runs, so a missing device stays a performance event.
- **The precision contract is stated per operation.** A `DeviceOp` declares its precision:
  - `PrecisionFloat32Rounded` means its device and CPU halves return the same bits. An `nn` test helper, `nn.CheckDeviceOp`, runs both halves on given inputs and compares every bit of the outputs and gradients. That claim is the caller's to prove; Insyra gives the tool and the exact library to make it true.
  - `PrecisionFloat32` means the device half may differ from the CPU half by float32 rounding. A `PrecisionFloat32Rounded` tape refuses such an operation unless the tape is told to accept device-dependent operations (`Tape.AllowDeviceDependent()`). The report then names the step as device-dependent, so the bit-identity claim is never made for a step it does not hold for.
- **Tensors expose and adopt device storage.** `Tensor.DeviceBuffer()` lends a resident tensor's buffer to a kernel without copying it. `nn.NewDeviceTensor(buffer, shape)` wraps a buffer a kernel wrote as a tensor the tape owns and releases.
- Nothing about existing operations, their kernels or their results changes.

## Capabilities

### New Capabilities

- `accel-custom-kernels`: caller-supplied WGSL kernels on resident buffers, batched into one submission, with bit-pattern buffers.

### Modified Capabilities

- `nn-device-tensors`: a tape runs a caller's `DeviceOp` where its inputs live, with a required CPU half, a declared precision and a parity check; tensors lend and adopt device buffers.

## Impact

- New: `accel/custom_kernel.go`, `accel/internal/wgpu/custom.go` (one-submission dispatch of several kernels), `nn/device_op.go`, and their tests, including a hardware-gated parity test.
- `Docs/accel.md` and `Docs/nn.md` gain the two APIs and the contract. Both CHANGELOGs gain entries.
- No breaking change. The `gogpu/wgpu` types stay internal: no public signature names them.
- Consumers: CoImNet's opt-in whole-step GPU path (its B5), built on this in its own repository.
