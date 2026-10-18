# Tasks

- [x] Reorder the row loop of `matMul2DWithWorkers` (`nn/kernels.go`).
- [x] Test: serial and parallel output against a per-element dot-product reference on seven shapes, including an empty inner dimension and inputs holding signed zeros, infinities, NaN and subnormal products (`nn/matmul_order_test.go`).
- [x] `BenchmarkMatMul2D`, before and after recorded in the proposal.
- [x] `go test ./nn/... ./ml/... ./accel/...` and `golangci-lint run ./nn/` pass.
- [x] Entries under `## Unreleased` in both CHANGELOGs.
- [x] `openspec validate matmul-row-order --strict` passes.
