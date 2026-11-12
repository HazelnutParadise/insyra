# 0026 — `insyra/parallel` stays, as groups that are run and awaited

**Status:** accepted
**Date:** 2026-09-13

## Context
The API review advised deprecating `insyra/parallel` in favour of `sync.WaitGroup` (#271).

## Decision
`insyra/parallel` stays. It exists so people who would not write wait groups can run data computations at once, and its `GroupUp(...).Run().AwaitResult()` shape is deliberate. The owner rejected the deprecation on 2026-09-13. A group is a reusable list of functions, and each `Run` returns a separate run that can only be awaited, so awaiting before running does not compile and running twice gives two independent results. `AwaitResult` returns the error directly (`parallel-runs-and-worker-errors`).

## Options rejected
- Deprecate the package and point users at `sync.WaitGroup`. Rejected by the owner.
- Report a worker's error through an `Err()` method on the run. It can be forgotten; returning it directly costs the one-line `a, b := r1.AwaitResult(), r2.AwaitResult()`.

## Consequences
A defect in `parallel` is fixed inside its design, not by retiring it.
