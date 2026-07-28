## Context
The seam was shaped for one operation over one column, which is all `OpSum` needed. Everything downstream inherited that shape: the backend receives `Values []float32`, `internal/wgpu.Sum` takes one slice, and the executor loops columns at the runtime level.

## Goals / Non-Goals
- Goals: one device submission per dataset; a seam that can express a multi-column kernel
- Non-Goals: a matrix-shaped result type, a second operation, or any change to precision policy — those arrive with the kernel that needs them

## Decisions

- Decision: `ExecuteRequest.Columns []ExecuteColumn` replaces `Column`/`Values`, and `ExecuteResponse.Reductions map[string]float64` replaces `Value`.
  - Rationale: the map keyed by column name is what `ExecutionResult` already exposes, so the runtime stops re-keying and the backend states its results in the same vocabulary the caller sees. A slice of columns preserves order for kernels that care about column position, which a map alone would lose.

- Decision: the response reports one set of cost figures per submission, not per column.
  - Rationale: transfer, dispatch and readback are properties of the submission. Attributing them per column would require the backend to invent a split. The runtime stops summing per-column costs and records what the backend measured.

- Decision: `internal/wgpu` uploads each column into its own buffer and dispatches per column inside one `Sum` call, rather than concatenating columns into one buffer.
  - Rationale: concatenation would need offset plumbing in the shader for a reduction that gains nothing from it — each column's partials still have to stay separate. Keeping buffers per column while collapsing the round trip is where the measured win is: one map/readback wait instead of one per column. Concatenation becomes worth revisiting when a kernel actually reads across columns.

- Decision: an empty column list is a runtime error, not a silent success.
  - Rationale: reaching the backend with nothing to do means the caller built a request wrong. The zero-row case is already handled above the seam.

## Measured

Apple M3, 262,144 rows per column, one `ExecuteDataTable` per iteration. "before" is one request per column, "after" is one submission with every column's dispatches and copies in a single command buffer landing in a single staging buffer.

| Columns | readback before | readback after | op before | op after |
| --- | --- | --- | --- | --- |
| 1 | 0.63 ms | 0.57 ms | 4.0 ms | 4.3 ms |
| 4 | 2.30 ms | 1.08 ms | 15.9 ms | 13.6 ms |
| 8 | 5.52 ms | 1.25 ms | 36.8 ms | 24.2 ms |

Readback scaled with column count before and is close to flat after — 4.4x less at eight columns, and 1.5x off the whole operation. A single column is unchanged within noise, which is the expected shape: the saving is the map wait that used to repeat per column.

The first implementation of this change collapsed only the runtime-level loop and left `internal/wgpu` mapping once per column, which measured no improvement at all. The batching had to reach the command buffer to matter.

## Risks / Trade-offs
- Risk: a third-party backend implementing the old interface stops compiling.
  - Accepted: the interface is days old, shipped in an unreleased branch, and the compiler reports the break precisely.
