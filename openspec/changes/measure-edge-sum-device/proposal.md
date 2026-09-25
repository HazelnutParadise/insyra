# Proposal: measure-edge-sum-device

## Why

#379 step 2b. The operating contract forbids a device kernel for an operation until measurement shows the device wins, measured against the all-core CPU. `nn-edge-sum-all-cores` recorded that CPU: 2.1–3.9× faster than one core, bound by memory.

Speed is not the only question. `EdgeSum` rounds every product to float32 before adding it, so arm64 and amd64 agree bit for bit; without the rounding arm64 fuses the multiply and the add and the bits change. `ENG.md` records that device MatMul matches the CPU only because Metal and Go on arm64 both fuse. A device `EdgeSum` must match the unfused CPU order, and whether a Metal shader can be kept from fusing is unknown. If it cannot, the choice between CPU agreement across platforms and device agreement goes to the owner.

## What Changes

- A prototype edge-sum compute kernel and a measurement harness in `accel/internal/wgpu`'s test files. It is not wired into `nn`, not exported, and not a production path. The kernel runs one invocation per output element and adds that node's incoming edges in ascending edge index, the order the CPU contract fixes.
- Two variants of the accumulation: the plain `acc + w * v`, and one written to keep the multiply and the add apart. Each is compared bit for bit, and in ULPs, against two CPU references: the contracted unfused order, and the same loop fused as arm64 compiles it.
- Timing over the sizes of `BenchmarkEdgeSum` (10k to 1M nodes, 10 or 100 edges per node, batch 1 and 16), best of five, in two situations: everything uploaded on every call, and the topology uploaded once with only weights and values sent per call. Upload, dispatch and readback are all included.
- The verdict is recorded in `delivery-status.md`: device against CPU per size, and which variant, if any, matches the unfused order. A device that loses, or cannot match without the owner changing the order, is a complete result of this change.

## Non-Goals

- No production kernel, no `nn` wiring, no API change, no new dependency.
- No reverse-rule kernels. The forward kernel decides the question, because the value gradient has the same gather shape and the weight gradient is a per-edge product.
- No resident tensors (step 2c), which is an `accel` architecture change for the owner to decide.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `nn-edge-sum`: a device path for the operation is gated on this measurement.

## Impact

- `accel/internal/wgpu` test files only, plus `delivery-status.md`.
- Runs only with `INSYRA_ACCEL_GPU_TESTS=1` on a machine with a device; CI skips it.
