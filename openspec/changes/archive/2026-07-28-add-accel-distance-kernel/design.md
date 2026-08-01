## Context
The seam carries one dataset and returns one number per column. A distance kernel needs a second operand — the query points — and returns a matrix, so both halves have to widen. The measured parity rule from the delivery plan applies: the CPU reference must be written in the natural `acc + diff*diff` form, because Metal and Go on arm64 both contract that into a fused multiply-add and contract it the same way.

## Goals / Non-Goals
- Goals: one kernel where the device wins on measurement, held to bit parity against its CPU reference
- Non-Goals: wiring `stats.KMeans` or the KNN functions to it — the assignment step, the centroid update loop and the interface adaptation are their own slice; `float64` distances, which no WebGPU backend can compute

## Decisions

- Decision: the operation computes distances to a set of query points rather than the full all-pairs matrix.
  - Rationale: all-pairs output grows with the square of the row count — a 100,000-row dataset would produce 40 GB. Every consumer named above actually wants rows against a small set: centroids for KMeans, the test set for KNN. The output is then rows times queries, which stays proportional to the data.

- Decision: one thread per (query, row) pair, with the dimension loop sequential inside it.
  - Rationale: it makes the operation order identical to the CPU reference by construction, which is what parity needs. No cross-thread reduction means no associativity question at all — the parity measurement was taken on exactly this shape.

- Decision: the CPU reference is written `acc = acc + diff*diff`, deliberately not `acc + float32(diff*diff)`.
  - Rationale: measured on this host, the fused form is bit-identical to the GPU across 4096 outputs and the explicitly-rounded form differs in 1137 of them by up to 2 ulp. Writing the "safe-looking" rounded form would guarantee divergence. A comment records this, because the code looks like it is missing a precaution.

- Decision: the parity test is a normal test that runs on whatever host it is on, and reports rather than assumes.
  - Rationale: Go emits FMA on arm64 but not on amd64, so a host where the CPU stops fusing while the GPU keeps fusing would break parity. That is a property of the platform, not of the kernel, and the honest way to handle it is to measure it there. An operation that fails parity on a host must stay on the CPU there once default-on lands.

- Decision: distances are returned as a flat `[]float32` indexed query-major.
  - Rationale: it is the device's natural layout, it avoids allocating a slice header per query, and the consumers that will read it are looping over queries anyway. Returning `float64` would imply a precision the kernel does not have.

## Measured

Apple M3, 100,000 rows, 16 dimensions, sweeping the query count. Both sides compute the same rows-by-queries output.

| Queries | CPU | GPU | readback | ratio |
| --- | --- | --- | --- | --- |
| 1 | 2.57 ms | 6.96 ms | 4.27 ms | 0.37x — GPU loses |
| 4 | 4.97 ms | 5.36 ms | 2.97 ms | 0.93x |
| 16 | 16.0 ms | 9.56 ms | 7.33 ms | 1.7x |
| 64 | 58.9 ms | 18.3 ms | 16.1 ms | 3.2x |

The parity gate passes: 160,000 values bit-identical at 16 queries, and 6,400,000 bit-identical at 64 queries across multiple dispatches.

The win is real but far smaller than the 62x the arithmetic-intensity spike recorded for pairwise work, and the reason is the output. The spike reduced its pairwise matrix down to one value per row; this operation materialises the whole rows-by-queries matrix, so readback grows with the answer and dominates — 16 ms of the 18 ms at 64 queries. Compute is not the constraint; moving the result is.

That points at the shape the consumers actually want. `stats.KMeans` needs the nearest centroid per row, `stats.KNNClassify` needs the k smallest per row — both collapse the matrix on the device and return one value per row instead of one per pair. Folding the argmin into the kernel is the next slice, and the numbers above are the argument for it.

Two corrections were made during implementation, both worth recording. An earlier sweep reported 28x at 64 queries; the benchmark was not checking `Accelerated`, and the kernel had silently fallen back because 6.4 million outputs need about 100,000 workgroups against a limit of 65,535. The dispatch is now split, and the benchmark fails rather than mistiming a fallback. The first attempt at splitting reused one uniform buffer and rewrote it per slice, which does not work: `WriteBuffer` is a queue operation and does not interleave with recorded commands, so every dispatch saw whichever base was written last. Each slice now has its own params buffer and bind group.

## Risks / Trade-offs
- Risk: the request and response grow fields that only one operation uses.
  - Accepted, and preferred to a second seam. `Op` already selects the shape; a backend that does not implement an operation says so, which the executor already handles.
- Risk: parity is verified on one platform.
  - This is the point of making it a test rather than a guarantee. The gate travels with the code.
