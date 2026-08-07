# Change: First real GPU execution

## Status
Closed on 2026-08-01 as superseded, with one task never completed. The archive records what actually happened rather than the plan.

**What it achieved.** It took the accel runtime from a scaffold verified against stubs to one that executes on real hardware, and it established the practices the rest of the phase ran on: a CPU reference for every device operation, bit-parity asserted rather than assumed, measured transfer and dispatch and readback timings replacing fabricated per-backend constants, and observable fallback reasons.

**What was superseded.** The operation it added — a column reduction on a device — was removed on 2026-07-29, measured at 0.7x against the CPU. Memory-bound work moves one value per element and performs one addition on it, so the transfer costs more than the arithmetic it feeds, on every device, permanently. Its spec delta therefore describes behaviour that no longer exists, which is why this change is archived without applying it: the main `accel-gpu-execution` spec already reflects the code as it stands.

**What was never done.** Task 1.13 required the numeric test to pass on a non-Apple host before archiving. It never ran: this project has no Windows or Linux machine with an NVIDIA or AMD GPU, and none became available. The backend is verified on macOS and Metal only, and that limitation is real — it is carried forward as a follow-up in `AGENTS.md` rather than buried here. Task 1.14 assumed a second operation would exist to fit the profitability rule against both ends of the arithmetic-intensity range; the operations that existed were removed, so its premise is gone with them.

## Original proposal

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
