# Proposal: thread-ccl-eval-depth

## Why

Issue #191 measured that CCL evaluation spends most of its time on recursion-depth bookkeeping rather than on evaluating: every AST node of every row pays a `goid.Get()`, two `sync.Map` operations, and a deferred closure — twice when a function call is involved. Removing that bookkeeping outright makes the representative expression 4.8x (10k rows: 15.05ms → 3.11ms) to 6.2x (100k rows: 141.59ms → 22.76ms) faster; that number is the upper bound a cheaper guard can chase. The guards themselves must stay — they stop pathological or re-entrant expressions from overflowing the stack.

## What Changes

- Recursion depth becomes an explicit parameter threaded through the evaluator's call stack: `evaluateWithContext` and every recursive helper pass `depth+1`, public entry points start at 0, and the limit check compares one register against `maxEvalDepth`. The function-call guard in `callFunction` gets the same treatment with its own counter and limit. Both limits (10000 / 20) and both error messages stay exactly as they are.
- The two process-global `sync.Map`s (`evalDepthByGoid`, `funcCallDepthByGoid`), their reset helpers, and the `goid` import disappear from `internal/ccl`.
- **Correction to the issue's claim**: threading the depth does *not* remove the `petermattis/goid` dependency from `go.mod` — `internal/core/atomic.go` also uses it for the actor model. The dependency stays; only CCL's hot-path use of it goes away. The change records this so nobody deletes the dependency on the issue's say-so.
- A benchmark records how much of the measured 4.8x–6.2x upper bound the parameter version recovers, on both row counts.
- Depth-limit tests are written before the refactor and must still trip both guards after it — equal strictness is demonstrated, not assumed.
- Both changelogs carry the user-visible entry; the old "CCL transform chains 11x on GPU" measurement is flagged as invalidated in `delivery-status.md`, since it was taken against this interpreter single-core.

## Capabilities

### New Capabilities

- `ccl-evaluation`: recursion-depth guarding in the CCL evaluator — strict limits enforced through stack-local state, with no process-global goroutine-keyed bookkeeping on the per-node hot path.

### Modified Capabilities

None — no existing spec covers CCL evaluation.

## Impact

- **Code**: `internal/ccl/ccl_evaluator.go` (~23 recursive call sites plus entry points and comparison/fold helpers), `internal/ccl/ccl_functions.go` (`callFunction`, single call site in the evaluator), deletion of both depth maps and reset helpers.
- **API**: none — exported entry points and `MapContext` keep their signatures; results and error semantics unchanged.
- **Dependencies**: `goid` removed from `internal/ccl` imports only; stays in `go.mod` for `internal/core`.
- **Docs**: `CHANGELOG.md` / `CHANGELOG_TW.md` (Core section). CCL docs describe syntax, not the guard mechanism — checked, and touched only if they mention it.
