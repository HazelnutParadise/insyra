# Design: thread-ccl-eval-depth

## Context

Both CCL depth guards are implemented the same way: `goid.Get()` keys a process-global `sync.Map` holding the current goroutine's depth, incremented on entry and decremented in a deferred closure. `evaluateWithContext` ([ccl_evaluator.go:233](../../../internal/ccl/ccl_evaluator.go)) pays this per AST node per row; `callFunction` ([ccl_functions.go:67](../../../internal/ccl/ccl_functions.go)) pays it again per function call. Issue #191's A/B measurement puts the recoverable cost at 4.8x–6.2x on a representative ten-node expression.

Facts that bound the design:

- The two guards are independent counters with independent limits: `maxEvalDepth = 10000`, `maxFuncCallDepth = 20`. They do not share state today.
- `callFunction` has exactly one call site, in the evaluator's `funcCallNode` case. Arguments are evaluated *before* the call, so within pure CCL, `callFunction` frames do not nest — its counter can only exceed 1 when a registered custom `Func` re-enters CCL on the same goroutine.
- `checkASTDepth` ([ccl_compiler.go:48](../../../internal/ccl/ccl_compiler.go)) already rejects ASTs deeper than `maxEvalDepth` at compile time, so the runtime guard is a second layer, not the only layer.
- `goid` is also used by `internal/core/atomic.go`; the dependency cannot leave `go.mod` in this change.

## Goals / Non-Goals

**Goals:**

- Recover most of the measured 4.8x–6.2x upper bound, with the recovery benchmarked and recorded, not asserted.
- Both guards stay exactly as strict: same limits, same error messages, demonstrated by tests that trip each limit before and after the refactor.
- Concurrent evaluations remain independent — now by construction (stack-local state) rather than by goroutine-ID bookkeeping.
- `internal/ccl` drops its `goid` import and both `sync.Map`s.

**Non-Goals:**

- No change to CCL syntax, semantics, results, or exported signatures (`Evaluate*`, `MapContext`, `RegisterFunction` families).
- No removal of `goid` from `go.mod` — `internal/core` still uses it.
- No touch of `checkASTDepth` or the parser's `maxParseDepth`; the compile-time layer already does its job.
- No revisiting of the limits' values (10000 / 20); this change moves the mechanism, not the policy.

## Decisions

1. **Plain `int` parameters, one per guard.** `evaluateWithContext(n, ctx, depth)` passes `depth+1` at every recursive site — including the comparison and fold helpers that recurse on its behalf — and every public entry starts at 0. `callFunction(name, args, callDepth)` carries its own counter along the funcCallNode path. Alternatives rejected: `context.Context` values (interface conversion and allocation on the exact hot path being cleaned); a shared eval-state struct (two independent ints don't justify a carrier type or the pointer chase).
2. **The func-call guard is kept, parameterized, even though pure CCL cannot nest it.** Analysis says the counter only exceeds 1 via re-entrant custom `Func`s; with stack-threading, such a re-entry starts a fresh count at the public entry instead of accumulating per goroutine. That is a semantic change confined to a pathological case — a custom function recursing through CCL more than 20 frames deep no longer trips at frame 20 of *accumulated* re-entries, but each entry still carries the full protection of both guards and the compile-time depth check. This is recorded here deliberately: the per-goroutine accumulation was an artifact of the implementation, not a documented contract, and preserving it is what the whole `sync.Map` existed to do.
3. **Strictness is pinned by tests written before the refactor.** One test trips the eval guard (an AST built directly, since the compiler's own depth check blocks the parse path), one trips the func-call guard (via the internal seam if unreachable from public API). Both must fail with the existing messages before the refactor and after it — the fix-shown-to-fail discipline pointed at behavior that must *not* change.
4. **The reset helpers are deleted, not adapted.** `internal/ccl`'s per-goroutine reset functions exist only to clean the maps; with stack-local state there is nothing to reset. Their call sites (tests or defensive callers) are removed with them.
5. **Benchmark records recovery against the same-session upper bound.** The A/B from #191 (guards removed entirely) is re-measured on this host in the same run as the parameter version, so the recovery fraction compares like with like instead of quoting the issue's numbers across machines.

## Risks / Trade-offs

- [~23 recursive call sites must all thread `depth+1`; missing one silently weakens the guard] → the pre-written limit tests catch a broken guard; `go vet` and the compiler catch a missed signature. The single-file blast radius keeps review tractable.
- [Re-entrant custom `Func` semantics change (Decision 2)] → recorded explicitly; the guard still exists per entry, the compile-time check still bounds AST shape, and no known caller registers re-entrant functions.
- [Recovery falls short of the upper bound] → the benchmark task records the actual number; even half the measured 4.8x is the largest single CPU win in the acceleration survey, and the mechanism carries no threshold to tune.
- [Concurrent-evaluation regression] → a race-detector test evaluates in parallel goroutines; stack-local state cannot interfere by construction, and the test proves the wiring didn't reintroduce sharing.

## Open Questions

- None blocking. The only judgment call — what to do with the func-call guard's cross-re-entry accumulation — is decided above (Decision 2) rather than left open.
