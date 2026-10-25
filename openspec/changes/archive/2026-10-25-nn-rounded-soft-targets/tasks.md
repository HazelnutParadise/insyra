# Tasks

- [x] Host sequences in `nn/soft_ce_rounded.go` with an independent-oracle test (`nn/soft_ce_rounded_test.go`).
- [x] Device kernels in `accel/internal/wgpu/soft_ce.go`, the hard-label row kernel on `log1p`, wrappers in `accel/resident.go`, wiring in `nn`.
- [x] Device against host, bit for bit (`nn/soft_ce_device_test.go`, gated by `INSYRA_ACCEL_GPU_TESTS=1`).
- [x] `Docs/nn.md`, entries under `## Unreleased` in both CHANGELOGs.
- [x] `openspec validate nn-rounded-soft-targets --strict` passes.
