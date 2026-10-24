# Tasks

## 1. accel: caller kernels and one-submission dispatch

- [ ] 1.1 Refactor `accel/internal/wgpu` so `kernel.run` is one dispatch of a multi-dispatch path that builds every bind group and uniform under one lock, encodes all passes into one command encoder and submits once; Insyra's own kernels keep their results bit for bit.
- [ ] 1.2 `accel.KernelSpec`, `accel.NewKernel`, `accel.Dispatch` and `accel.Run`, with the refusals of the spec (counts, released, host, mixed devices, binding limit, empty name or source).
- [ ] 1.3 `accel.NewDeviceBufferBits` and `DeviceBuffer.BitsToHost`, budgeted by bytes.
- [ ] 1.4 Tests: host refusals; gated hardware tests for a single kernel, a chained `Run`, bit round-trip, and a `Library` exact dot product equal to `RoundedMatMul`.

## 2. nn: DeviceOp on the tape

- [ ] 2.1 `nn.DeviceOp`, `nn.VJP` and `Tape.Apply`: location check, half selection, VJP recording, output ownership, gradient-location check.
- [ ] 2.2 `Tape.AllowDeviceDependent`, `Tape.DeviceDependent`, and the report entry for a `PrecisionFloat32` operation.
- [ ] 2.3 `Tensor.DeviceBuffer` and `nn.NewDeviceTensor`.
- [ ] 2.4 `nn.CheckDeviceOp`.
- [ ] 2.5 Tests: refusals; host-only run through the CPU half; gated device run with `Backward` that reads nothing back; `CheckDeviceOp` on a faithful and on a deliberately wrong pair.

## 3. Documentation and verification

- [ ] 3.1 `Docs/accel.md` and `Docs/nn.md`: the two APIs, the binding layout, the precision tiers and how to prove a Rounded claim; Traditional-Chinese counterparts where they exist.
- [ ] 3.2 ENG.md: callers' kernels extend the exact library under the same rule; a `PrecisionFloat32` operation is labelled, never silently mixed into a Rounded step.
- [ ] 3.3 Entries under `## Unreleased` in both CHANGELOGs.
- [ ] 3.4 `go vet ./...`, `golangci-lint run ./...`, `go test ./...` in the core module and in `accel/backend/wgpu`; `INSYRA_ACCEL_GPU_TESTS=1` on the M3.
- [ ] 3.5 `openspec validate accel-custom-kernels --strict`.
