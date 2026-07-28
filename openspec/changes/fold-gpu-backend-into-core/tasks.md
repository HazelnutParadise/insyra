## 1. Implementation
- [x] 1.1 Validate the change with `openspec validate fold-gpu-backend-into-core --strict`
- [x] 1.2 Move the gogpu mechanics to `accel/internal/wgpu` with a neutral surface that does not import `accel`, keeping the software-adapter refusal and the unified-memory classification with their tests
- [x] 1.3 Add `accel/backend_wgpu.go`: map the neutral surface onto `Device` / `ExecuteRequest` / `ExecuteResponse` and register the probe and executor from `init`
- [x] 1.4 Delete the `accel/backend/wgpu` module and fold its requirements into the core `go.mod`
- [x] 1.5 Move the hardware test and benchmarks into `accel`. The hardware test stays gated behind `INSYRA_ACCEL_GPU_TESTS=1`; the benchmarks now skip themselves when no device is discovered, since the runtime owns device acquisition
- [x] 1.6 Confirm a consumer that imports only non-accel packages still builds without gogpu being compiled
- [x] 1.7 Update `Docs/accel.md`, README and README_TW, CHANGELOG and CHANGELOG_TW, and the CLI skill reference to drop the second install and the blank import
- [x] 1.8 Run `go test ./...`, `go test -race ./accel/...`, and the GPU test with `INSYRA_ACCEL_GPU_TESTS=1`
- [x] 1.9 Reverse the logged decision in `delivery-plan.md`, recording what measurement changed it
