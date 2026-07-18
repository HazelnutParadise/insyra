# Flatten CCL operator chains (n-ary fold)

## Why

`bound-ccl-compile-recursion` capped expression-tree depth at 10000 to stop fatal stack overflows. For *nesting* (parens, nested calls) that cap costs nothing — no real formula nests 10000 deep. But for *left-associative operator chains* (`A+B+C+...`) the cap is a real ceiling: machine-generated formulas can legitimately exceed 10000 terms, and the deep tree shape is purely an artifact of how the parser represents left association. The chain itself is semantically a flat left fold.

## What Changes

- New AST node `cclFoldChainNode{init, ops, operands}` representing a flattened left fold: `((init ops[0] operands[0]) ops[1] operands[1]) …`.
- The parser's precedence-climbing loop accumulates consecutive general operators into one fold node instead of nesting binary nodes. Chains of any length now produce depth-2 ASTs, so they clear the depth check, and evaluation walks them iteratively — chain length becomes RAM-bound instead of capped.
- Exclusions that preserve behavior exactly:
  - A **single** operator still produces `cclBinaryOpNode` — zero structural or perf change for the overwhelmingly common case.
  - `.` and `:` always stay binary (their evaluation is node-level special-cased: `evaluateRowAccess`/`evaluateRange`, plus `IsRowDependent`'s static-range logic); hitting one flushes the accumulator.
- All node-type switches gain a fold case: evaluator, `Bind`, `IsRowDependent`, `containsRowIndex`, `exceedsDepth`.
- Behavior-preservation test suite (differential + golden matrix + structural invariants) and compile/eval benchmarks with a before/after comparison gate: the change lands only if common-case performance does not regress (user condition).
- `Docs/CCL.md` Limits section updated: chains are no longer depth-capped; nesting still is.

## Capabilities

### Modified Capabilities
- `ccl-compile-limits`: the complexity cap now applies only to genuine nesting; left-associative operator chains of arbitrary length compile and evaluate without hitting it.

## Impact

- **Code**: `internal/ccl/ccl_types.go`, `ccl_parser.go`, `ccl_evaluator.go`, `ccl_compiler.go`; tests in `internal/ccl/` and `datatable_ccl_test.go`; new benchmark file.
- **Behavior**: strictly additive — inputs that previously worked produce identical values, identical error messages, and identical evaluation/error order (`&&`/`||` remain non-short-circuiting; left-to-right operand evaluation preserved). Inputs that previously failed with `expression too complex` (chains > 10000 terms) now succeed; that is the goal.
- **Perf**: eval of long chains gets faster (one recursion-depth bookkeeping op per chain instead of per node); simple expressions unchanged (single-op keeps the binary path). Verified by benchmarks.
- **Risk**: a missed fold case in a node-type switch, or an eval-order deviation. Mitigated by the differential test harness (fold vs unfolded evaluation equality, including error strings) and the full existing suite.
