## Context

The `dl` package executes float32 tensor kernels as plain Go functions. Its
`MatMul` and `Conv` implementations currently compute every output serially,
although each output element is independent of the others. The change must
improve the CPU baseline before device inference is considered, while keeping
the existing arithmetic contract: the reduction inside one output must retain
its serial order exactly.

## Goals / Non-Goals

**Goals:**

- Partition independent output work for 2-D MatMul, broadcasted batched
  MatMul, and Conv across `runtime.NumCPU()` workers.
- Keep small workloads serial and make parallel and serial outputs
  bit-identical.
- Preserve the public API, existing parity harnesses, and dependency-free
  pure-Go execution.

**Non-Goals:**

- Reordering reductions through K-blocking, lane-split accumulation, SIMD, or
  any other numerical transformation.
- Parallelizing other `dl` kernels or adding device kernels, cgo, or assembly.
- Adding a persistent worker-pool API or exposing worker-count controls.

## Decisions

1. **Use one local worker helper.** A small `sync.WaitGroup` helper divides a
   contiguous range into at most `runtime.NumCPU()` disjoint ranges. The
   2-D path assigns output rows, the batched path assigns batch-row pairs, and
   Conv assigns batch-output-channel-output-row groups. Each worker writes only
   its assigned output range, so no reduction or lock is needed.

2. **Keep the serial loop body as the reference implementation.** The public
   kernels select the worker count, while internal worker-aware functions also
   accept one worker for exact tests. The innermost MatMul and Conv loops stay
   in their original order, including float32 MatMul accumulation and Conv's
   float64 accumulation before narrowing.

3. **Gate dispatch by measured multiply-accumulate work.** A single threshold
   of `100_000` MACs keeps dispatch serial at or below the cutoff. It was
   selected from quick best-of-five crossover measurements on the 8-core M3,
   with margin for both MatMul and Conv. The helper also returns serial for
   empty or degenerate dimensions and caps workers at the number of output
   jobs.

4. **Prove parity through internal seams.** Tests call the same kernel logic
   once with one worker and once with `runtime.NumCPU()` on above-threshold
   2-D MatMul, broadcasted batched MatMul, and grouped, dilated Conv inputs.
   They compare tensor shapes and every float32 bit pattern exactly, then
   verify the public entry points use the same result.

## Risks / Trade-offs

- **[CPU frequency and host load can make wall-time measurements vary]** →
  Record the required benchmark's best-of-five result and use an independent
  same-process kernel probe to distinguish scheduling noise from kernel
  regressions. Do not treat a single wall-time run as a stable guarantee.
- **[Poor range balance could leave cores idle on small output shapes]** →
  Partition over batch-row or batch-channel-row jobs and cap workers to the
  available jobs; workloads below the measured threshold remain serial.
- **[A future edit could accidentally reorder a reduction]** → Keep the
  serial inner loops visible in the worker-aware functions and retain exact
  worker-one versus all-worker tests alongside the parity suite.

## Migration Plan

No data or public API migration is required. Existing callers use the same
`MatMul` and `Conv` functions and automatically receive serial execution for
small inputs and all-core execution for large inputs. Reverting the change
removes the worker helper and its tests without affecting persisted state.

## Open Questions

None for this change. Device execution and any broader CPU kernel parallelism
remain separate follow-up work.
