## Context

`ExecuteNearestExact` is a two-sided operation. The device can produce a
shortlist, but the host verifies candidates and resolves the final answer in
`float64`. The no-device path performs the same exact search entirely on the
host. Both paths were serialized on one core, and the shortlist's candidate-
major layout made the verification loop stride across memory.

The measured comparison must use the parallel host, not one core. Otherwise
the device appears faster simply because the baseline is artificially weak.

## Goals / Non-Goals

**Goals:**

- Parallelize exact host search and shortlist verification above a measured
  work threshold.
- Give each worker independent scratch storage and race-free result writes.
- Store and read shortlist candidates row-major.
- Recalibrate profitability from work per row against the multicore host.
- Preserve exact results, fallback behavior, and the recheck count.

**Non-Goals:**

- Changing the device's candidate ranking precision or final host decision.
- Making every small workload parallel.
- Introducing heterogeneous multi-device sharding.
- Claiming a speedup without rerunning the shape map against the parallel CPU.

## Decisions

### Split independent row ranges with a work threshold

The exact host loop is divided into contiguous row ranges. Below a measured
threshold it stays serial to avoid goroutine and scratch allocation overhead;
above it, workers receive disjoint ranges and private scratch buffers. The
same splitter is used for the no-device reference and device-shortlist
verification so the two paths retain the same computational contract.

Sharing scratch between workers was rejected because it would require locks in
the hot loop and would make race-free verification harder to prove.

### Write the shortlist row-major

The backend and CPU shortlist producer write each row's candidates contiguously.
Verification then consumes one row at a time, matching the host decision loop's
access pattern. The reader and producer change together so the layout is not a
silent wire-format mismatch.

Keeping candidate-major storage was rejected because the host would continue
to stride across the complete candidate array for every row, leaving the
parallel computation bound by memory access.

### Calibrate profitability on work per row

The dispatch decision uses the amount of row-by-dimension-by-query work rather
than query count alone. The threshold and delivery-plan table are set from the
parallel host and the measured device phases, including shapes where the
multicore CPU wins.

This keeps acceleration opt-in to the shapes where the device actually pays
and avoids carrying the old single-core speedup claim forward.

### Keep host verification authoritative

Workers may inspect device candidates and increment an atomic recheck count,
but the final index and distance are still selected by the host's `float64`
comparison with the established lowest-index tie rule. Parallelism changes
where the work runs, not what answer is trusted.

## Risks / Trade-offs

- **[Risk] Parallel overhead hurts small inputs]** → Keep the serial path below
  the measured threshold and test both sides of it.
- **[Risk] Worker-local state is accidentally shared]** → Allocate scratch per
  worker, run the non-device path under `-race`, and assert results against the
  serial reference.
- **[Risk] Row-major layout disagrees between backend and reader]** → Update
  both producers and the consumer in one change and test random and
  adversarial shortlist contents.
- **[Risk] Recheck counts change under parallel scheduling]** → Use an atomic
  counter and compare the total with the serial/device reference tests.
- **[Risk] Recalibration overfits the recorded hardware]** → Keep the table and
  threshold observable in the delivery plan and treat new hardware as a reason
  to measure again, not as a reason to weaken exactness.

## Migration Plan

The API and result values are unchanged. The runtime automatically uses the
parallel host path when the work threshold is met. Existing callers do not
need a migration; rollback restores the serial loops and previous shortlist
layout together. Any rollback must also restore the matching measurement table.

## Open Questions

Future work may reduce host verification further or add multi-device sharding,
but both require new measurements. Neither changes the rule that the CPU
settles the exact answer.
