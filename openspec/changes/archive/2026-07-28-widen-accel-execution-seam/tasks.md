## 1. Implementation
- [x] 1.1 Validate the change with `openspec validate widen-accel-execution-seam --strict`
- [x] 1.2 Add a test asserting the backend receives one request naming every column, and confirm it fails against the current per-column loop
- [x] 1.3 Reshape `ExecuteRequest` to carry `Columns []ExecuteColumn` and `ExecuteResponse` to carry `Reductions map[string]float64`
- [x] 1.4 Build one request per dataset in `ExecuteProjectedDataset`, and record the backend's measured cost rather than summing per-column figures
- [x] 1.5 Update the wgpu adapter and `internal/wgpu.Sum` to reduce a set of columns in one submission
- [x] 1.6 Measure the round-trip saving on a multi-column DataTable and record it in `design.md`
- [x] 1.7 Run `go test ./...`, `go test -race ./...`, and the GPU test with `INSYRA_ACCEL_GPU_TESTS=1`
