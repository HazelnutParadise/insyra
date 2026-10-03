# Proposal: engine-ccl-registry-contract

## Why

`engine/ccl` is how code outside the module reaches CCL's function registry, and three things about it were wrong (#259, EN-1 and CCL-29):

- It forwarded `RegisterFunction` and `RegisterAggregateFunction` but not `RegisterSequenceFunction`, so a whole-column function such as a custom `LAG` could only be added from inside the module.
- `ResetEvalDepth` and `ResetFuncCallDepth` have done nothing since recursion depth moved onto the call stack (#191), and nothing told a caller so.
- Nothing said what the registry is: one per process or one per table, safe to share between goroutines or not, whether a name can be registered twice. `engine/README.md` told callers to guard registration with `sync.Once`, which the registry has not needed since it took a lock.

Checking the concurrency claim before writing it down found a gap. Registering an aggregate or sequence function writes the function and a mark that says the name is the caller's, which the streaming forms read to stand down for a built-in name the caller replaced. The two were written under separate locks, so a reader between them saw the caller's function with no mark: measured on 2026-10-03, a reader polling while another goroutine re-registered a name 20,000 times saw that state 40,236 times.

## What Changes

- `engine/ccl` gains `SeqFunc` and `RegisterSequenceFunction`, forwarding to the registry the evaluator reads.
- `ResetEvalDepth` and `ResetFuncCallDepth` are marked Deprecated and keep doing nothing; an `AGENTS.md` follow-up records their removal.
- The package doc, each `Register*` function, `Docs/CCL.md` and `engine/README.md` state how the registry behaves: one per process, safe from any number of goroutines, each registration taking effect as a whole, names matched in any letter case, a later registration replacing an earlier one including a built-in's, no removal.
- `RegisterAggregateFunction` and `RegisterSequenceFunction` write the function and its mark under one lock.

Out of scope: removing the type aliases of the internal AST (`CCLNode`, `Context`, `EvaluationResult`, `MapContext`). That removes public API, which is the owner's decision; the recommendation goes on #259.

## Capabilities

### New Capabilities

- `ccl-function-registry`: what `engine/ccl` lets a caller register and how the registry behaves.

### Modified Capabilities

(none)

## Impact

- Code: `engine/ccl/ccl.go`; `RegisterAggregateFunction` and `RegisterSequenceFunction` in `internal/ccl/ccl_functions.go`.
- Tests: `engine/ccl/registry_contract_test.go`, `internal/ccl/registry_atomic_test.go`.
- Docs: `Docs/CCL.md`, `engine/README.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`, `AGENTS.md`.
