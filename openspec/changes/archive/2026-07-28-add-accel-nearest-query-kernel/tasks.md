## 1. Implementation
- [x] 1.1 Validate the change with `openspec validate add-accel-nearest-query-kernel --strict`
- [x] 1.2 Add `OpNearestQuery`, the index result on `ExecuteResponse`, and the CPU reference seeded from the first query
- [x] 1.3 Add `Session.ExecuteNearestQuery` and `NearestQueryCPU`
- [x] 1.4 Implement the WGSL kernel: one thread per row, ties to the lowest index
- [x] 1.5 Add tests for hand-computed answers, tie-breaking, and bit parity against the reference
- [x] 1.6 Benchmark against the CPU reference and against `OpSquaredDistance`, and record the numbers
- [x] 1.7 Run `go test ./...`, `go test -race ./...`, every accel test individually, and the GPU tests
- [x] 1.8 Update `Docs/accel.md`, CHANGELOG and CHANGELOG_TW, and `delivery-plan.md`
