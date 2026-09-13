## Why

`parallel` lets someone who does not write goroutines and wait groups run several data computations at once: `parallel.GroupUp(f1, f2).Run().AwaitResult()`. The owner ruled on 2026-09-13 (#271) that the package and that shape stay. What the review found wrong sits inside the shape, measured the same day:

- `Run` on the same group twice ran every function again into the same result slots, and two goroutines calling `Run` together were a data race under `-race`.
- `AwaitResult` on a group nobody ran returned empty results at once, with nothing to say so.
- A function that panicked left `[]any{error}` in its slot, indistinguishable from a function that returned an `error` itself.
- `Docs/parallel.md`'s own example increments one `counter` from two goroutines, which is a data race, and says `AwaitNoResult` skips result collection, which `Run` never did.

## What Changes

- **BREAKING**: `(*ParallelGroup).Run` returns a new `*RunningGroup`. A `ParallelGroup` is a reusable list of functions with only `Run`; every `Run` starts an independent run with its own results. A `RunningGroup` has only `AwaitResult` and `AwaitNoResult`. Awaiting a group that was never run no longer compiles, and a run cannot be run again. `GroupUp(...).Run().AwaitResult()` keeps its spelling.
- **BREAKING**: `AwaitResult` returns `([][]any, error)` and `AwaitNoResult` returns `error`. A result slot holds only what its function returned, including any `error` the function returned itself.
- A function that panics, or a value that cannot be called (not a function, a nil function, a function that takes arguments), leaves its slot `nil` and is reported as a `*parallel.WorkerError` carrying its index, the panic value, the stack at the panic, or the reason it could not be called. Several are joined with `errors.Join` in index order. `errors.Is` reaches a panic value that is itself an error.
- `GroupUp` copies its arguments, so changing the caller's slice afterwards does not change the group.
- `mkt.RFM` and `stats.RepeatedMeasuresANOVA` check the error instead of dropping a panic.
- `Docs/parallel.md` is rewritten to match, the tutorial that uses `AwaitResult` is updated, and the package description in `AGENTS.md` stops calling it a map/reduce over tables.

Not in this change: a context or a concurrency limit (every function gets a goroutine, which is what the package is for), and a generic typed API (`any` is what lets functions with different return types share a group).

## Capabilities

### New Capabilities
- `parallel-groups`: what a group, a run, a result slot and a worker failure are in `insyra/parallel`.

### Modified Capabilities
None.

## Impact

- `parallel/parallel_computing.go`, `parallel/parallel_test.go`.
- `mkt/rfm.go`, `stats/anova.go`.
- `Docs/parallel.md`, `Docs/tutorials/python-enrichment-and-parallel-batch.md`, `AGENTS.md`, both CHANGELOGs.
- `api-review.md` P-1, P-2, P-4, P-5, `delivery-status.md`, issue #271.
- Outside the repository: code that assigns `AwaitResult()` to one variable, or names `*ParallelGroup` as the type of `Run()`, must change. idensyra's generated symbol table for `insyra/parallel` needs the new types.
