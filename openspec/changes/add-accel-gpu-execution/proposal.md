# Change: Execute a real GPU kernel from the accel runtime

## Why
`accel` can enumerate GPUs but cannot run anything on one. The only seam on the execution path, `BackendAllocator.Materialize`, takes no operation and returns three byte counts, and `ExecutionResult` has no field that could carry a computed value. The three builtin backend allocators divide a byte count by an invented constant and the CLI prints the result as `execution.bytes_moved`, so users are shown fabricated telemetry. No existing accel change requires compiling or dispatching a kernel, which means the cost model, the profitability rule, and the multi-device plan have never once been checked against a device.

The runtime needs one operation that produces a correct number on real hardware before any further backend surface is worth building.

## What Changes
- Add a new `accel-gpu-execution` capability
- Replace the report-only `BackendAllocator` seam with an execution seam that carries an operation, host input, and a returned result
- Adopt `github.com/gogpu/wgpu` as the first real backend, in a separate Go module so the core `insyra` module gains no GPU dependency
- Define `float32` eligibility, and refuse `float64` columns with an observable reason instead of silently downcasting them
- Remove the fabricated per-backend transfer constants and the telemetry derived from them, and replace them with measured transfer, dispatch, and readback timings

## Impact
- Affected specs: `accel-gpu-execution`
- Affected code: `accel/executor.go`, `accel/types.go`, `accel/allocator_builtin.go` (removed), new `accel/backend/wgpu` module, `cli/commands` accel output
