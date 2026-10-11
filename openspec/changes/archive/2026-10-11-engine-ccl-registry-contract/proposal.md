# Proposal: engine-ccl-registry-contract

## Why

`engine/ccl` is how code outside the module reaches CCL's function registry, and three things about it were wrong (#259, EN-1 and CCL-29):

- It forwarded `RegisterFunction` and `RegisterAggregateFunction` but not `RegisterSequenceFunction`, so a whole-column function such as a custom `LAG` could only be added from inside the module.
- `ResetEvalDepth` and `ResetFuncCallDepth` have done nothing since recursion depth moved onto the call stack (#191), and nothing told a caller so.
- Nothing said what the registry is: one per process or one per table, safe to share between goroutines or not, whether a name can be registered twice. `engine/README.md` told callers to guard registration with `sync.Once`, which the registry has not needed since it took a lock.

## What Changes

- `engine/ccl` gains `SeqFunc` and `RegisterSequenceFunction`, forwarding to the registry the evaluator reads.
- `ResetEvalDepth` and `ResetFuncCallDepth` are marked Deprecated and keep doing nothing.
- The package doc, each `Register*` function, `Docs/CCL.md` and `engine/README.md` state how the registry behaves: one per process, safe from any number of goroutines, each registration taking effect as a whole, names matched in any letter case, a later registration replacing an earlier one including a built-in's, no removal.

Left on 0.4: the concurrency fix that writes a caller's function together with the mark that makes its name the caller's under one lock; this line has no such marks.

Out of scope: removing the type aliases of the internal AST (`CCLNode`, `Context`, `EvaluationResult`, `MapContext`). That removes public API, which is the owner's decision.

## Capabilities

### New Capabilities

- `ccl-function-registry`: what `engine/ccl` lets a caller register and how the registry behaves.

### Modified Capabilities

(none)

## Impact

- Code: `engine/ccl/ccl.go`.
- Tests: `engine/ccl/registry_contract_test.go`.
- Docs: `Docs/CCL.md`, `engine/README.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
