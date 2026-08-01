## Context

The approximate nearest-query kernel ranks in `float32`, while the callers
that need its result compute in `float64`. Returning the device's winner would
therefore violate the no-number-change acceleration contract. Measurements
showed a viable boundary: the device can rank a small shortlist, and the host
can recompute that shortlist in `float64` while still beating the complete host
search for sufficiently large query sets.

The exact operation is consequently a two-stage selection. The device proposes
candidates and a boundary value; the CPU decides the answer from the original
data and remains the reference when no device runs.

## Goals / Non-Goals

**Goals:**

- Add a shortlist kernel and an exact nearest operation over it.
- Keep final indices and distances identical to `NearestExactCPU`.
- Recompute every row whose shortlist boundary is not trustworthy under the
  dimension-derived `float32` error bound.
- Preserve lowest-query-index tie breaking.
- Reject invalid `m`, report full-row rechecks, and decline unprofitable shapes.
- Keep the operation usable with and without a device.

**Non-Goals:**

- Returning a device `float32` answer as the final result.
- Using the narrowed device buffer for host verification.
- Hiding rechecks or claiming profitability below the measured crossover.
- Adding multi-device sharding or changing other distance operations.

## Decisions

### Make the device return a shortlist, not a winner

`OpNearestShortlist` carries the shortlist width, candidate indices and
distances, and the best rejected distance through the execution request and
response. The WGSL kernel keeps the smallest candidates per row and handles
rows above the workgroup limit through the same dispatch splitting as the other
kernels.

Returning only the winner was rejected because a single `float32` comparison
can drop the true `float64` winner near a tie, while a shortlist gives the host
enough candidates to settle the decision.

### Recompute from original host values in `float64`

The host recomputes shortlisted distances from the original dataset and query
values, never from the narrowed device copy. It applies the dimension-derived,
conservative unfused error bound to the shortlist boundary. A boundary inside
that bound triggers a full row recomputation against every query point.

This is the correctness boundary: the device may propose an incomplete list,
but no row with an untrusted boundary is allowed to return a device-derived
decision.

### Keep a complete CPU reference and observable economics

`NearestExactCPU` uses the same `float64` distance and tie rules and is used
when no device participates. `ExecuteNearestExact` reports `workload-not-
profitable` below the measured crossover and includes the number of fully
recomputed rows. These fields distinguish a correct CPU answer from a device
answer and make shortlist width and crossover assumptions measurable.

### Choose shortlist width from requested output

The operation rejects `m` larger than the query count and chooses a shortlist
wide enough for the requested `m` rather than using one global constant. This
keeps memory bounded for small results and leaves the exact fallback available
when the boundary is uncertain.

## Risks / Trade-offs

- **[Risk] The error bound is too optimistic]** → Use the conservative unfused
  form derived from dimension count and recheck the whole row whenever the
  boundary is within it.
- **[Risk] Rechecking every row removes the device benefit]** → Report the
  count, benchmark the crossover, and decline shapes where the full host path
  wins.
- **[Risk] Device and CPU tie rules disagree]** → Make the CPU reference's
  lowest-index rule authoritative and test duplicates and equidistant queries.
- **[Risk] The shortlist reader uses the wrong layout]** → Test the producer,
  backend response, and host verifier together, including large row counts.
- **[Risk] No device leaves callers without an answer]** → Route to
  `NearestExactCPU` and preserve fallback metadata rather than returning empty
  result slices.

## Migration Plan

This is additive. Existing approximate nearest-query callers keep their API;
new callers that require exact `float64` results use `ExecuteNearestExact`.
No persisted data changes. Rollback removes the exact operation and its kernel
while leaving the pre-existing approximate operation intact.

## Open Questions

The shortlist width and profitability threshold should be recalibrated when
backend hardware or host parallelism changes. Multi-device execution and a
different candidate-ranking kernel require separate measurements and changes.
