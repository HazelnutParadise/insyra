## Context

Measurements showed that the first reduction and distance kernels did not pay
for transfer, readback, or the CPU baseline used by the runtime. The nearest
`f32` query path was also superseded by the exact host-verification path. Keeping
those operations would preserve kernels, caches, backend dispatch, CLI surface,
and documentation for work the strategy explicitly says should stay on the
CPU.

The removal is safe for released users because these execution APIs were not in
the released `v0.3.0` surface. The exact-nearest operation and the runtime seam
remain the supported acceleration path.

## Goals / Non-Goals

**Goals:**

- Remove the measured-losing sum, all-pairs distance, and approximate nearest
  kernels and their unused runtime surfaces.
- Remove their backend dispatch, pipeline caches, reference helpers, and CLI
  command without leaving dead paths.
- Keep `OpNearestShortlist`, `ExecuteNearestExact`, discovery, cache, scheduler,
  fallback, and measurement tooling.
- Correct speedup documentation and the delivery plan to use the multicore CPU
  baseline.

**Non-Goals:**

- Replacing the removed kernels with unmeasured alternatives.
- Removing the exact nearest operation or its CPU verification path.
- Changing the public released API or the runtime's fallback contract.
- Declaring every memory-bound operation permanently ineligible without a new
  measurement and proposal.

## Decisions

### Remove the operation at every layer

The change removes the operation type, executor entry point, CPU reference,
backend method, shader, cache handle, and CLI command together. Leaving a
public type or a backend arm without a profitable caller would make the code
appear supported while preserving the maintenance cost.

Deleting only the CLI was rejected because library callers would still reach
the same losing kernels. Deleting only the shader was rejected because it would
leave an API that fails at runtime.

### Keep the exact selection-shaped path

`OpNearestShortlist` and `ExecuteNearestExact` stay because their result is a
selection and the host recomputes the final `float64` answer. The device
proposes candidates; the CPU decides, so the path has a correctness contract
that the approximate nearest result did not have.

### Rebase published measurements on the real CPU baseline

Documentation and changelogs use the measured multicore host path rather than
single-core numbers. This prevents a removed operation's old speedup claim from
surviving as an implied product promise.

## Risks / Trade-offs

- **[Risk] An unreleased branch caller imports a removed symbol]** → The
  removal is intentionally breaking for pre-release consumers; document the
  replacement exact operation where one exists and do not preserve a dead
  compatibility shim.
- **[Risk] A reference helper still has callers after deletion]** → Build,
  vet, and search the full repository for each removed symbol, including CLI
  and backend packages.
- **[Risk] A future workload would have benefited from a removed kernel]** →
  Require a new benchmark, multicore baseline, and OpenSpec change before
  reintroducing it.
- **[Risk] Documentation still implies broad GPU support]** → Update both
  changelogs, `Docs/accel.md`, and the delivery plan in the same change.

## Migration Plan

Released callers have no migration because the removed execution surface was
not released. Development callers should use the exact nearest operation where
they need a selection result. Rollback is a source revert, but any reintroduced
kernel must first restore a measured, reviewed proposal rather than merely
restoring deleted code.

## Open Questions

Whether another reduction or distance operation becomes profitable is a future
measurement question. The answer must be based on the multicore CPU and full
end-to-end device costs before a new kernel is proposed.
