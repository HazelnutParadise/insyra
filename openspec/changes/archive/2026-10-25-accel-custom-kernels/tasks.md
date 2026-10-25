# Tasks

## 1. accel: caller kernels and one-submission dispatch

- [x] 1.1 Route `kernel.run` through a shared multi-dispatch submit that builds every bind group and uniform under one lock, encodes all passes into one encoder and submits once; Insyra's own kernels keep their results.
- [x] 1.2 `accel.KernelConfig`, `NewKernel`, `Kernel.Bind`, `Binding`, `Dispatch`, `Submit` with a shared parameter ring, with the refusals of the spec.
- [x] 1.3 `NewBitsBuffer`, `BitsToHost` (read as bytes), `Write`, `WriteBits`, `Zero`, `DeviceBuffer.Released`, `ResidentLimits`.
- [x] 1.4 Host and hardware-gated tests.

## 2. nn: Custom on the device

- [x] 2.1 `SetPrecision` accepts `PrecisionFloat32`; built-in operations treat it as rounded.
- [x] 2.2 `Tape.Custom` accepts resident tensors on a `PrecisionFloat32` tape, adopts the output, owns the gradients.
- [x] 2.3 `Tensor.DeviceBuffer` and `nn.NewDeviceTensor`.
- [x] 2.4 Host and hardware-gated tests, including a tape step with a caller kernel compared with the host oracle.

## 3. Documentation and verification

- [x] 3.1 `Docs/accel.md`, `Docs/nn.md`, ENG.md, both CHANGELOGs.
- [x] 3.2 `go vet ./...`, `golangci-lint run`, `go test ./...`, the `INSYRA_ACCEL_GPU_TESTS=1` suites of `accel`, `accel/internal/wgpu` and `nn` on the M3.
- [x] 3.3 `openspec validate accel-custom-kernels --strict`.
