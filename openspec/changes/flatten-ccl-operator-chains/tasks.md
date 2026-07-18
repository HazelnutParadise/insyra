# Tasks — flatten CCL operator chains

## 1. Baseline

- [x] 1.1 Added `internal/ccl/ccl_bench_test.go` (compile + eval × simple/medium/chain-100/chain-5000) and recorded baseline on pre-change HEAD (5 runs each).

## 2. Implementation

- [x] 2.1 `ccl_types.go`: `cclFoldChainNode` with the `len(ops) == len(operands) >= 2` invariant documented.
- [x] 2.2 `ccl_parser.go`: precedence loop accumulates general operators with **deferred allocation** (scalar pending slot for the single-op case, slices only from the second op — added after benchmarks showed the naive always-allocate version cost +22% on simple compiles); `materializeFold` (0→as-is, 1→binary, ≥2→fold); `.`/`:` flush first and stay binary.
- [x] 2.3 `ccl_evaluator.go`: fold cases in `evaluateWithContext` (left fold via `applyOperator`, no short-circuit), `IsRowDependent`, `containsRowIndex`.
- [x] 2.4 `ccl_compiler.go`: fold cases in `Bind` and `exceedsDepth`.

## 3. Behavior-preservation tests

- [x] 3.1 `ccl_fold_behavior_test.go`: `unfoldChains` helper + 48-expression differential corpus — values, error strings, `IsRowDependent`, `containsRowIndex` identical between folded and unfolded shapes, both pre- and post-`Bind`.
- [x] 3.2 Golden order-sensitivity tests (`2^3^2`=64, `10-2-3`=5, `100/5/2`=10, boolean chains) + dedicated no-short-circuit tests (`false && (1/0 > 0)` must error).
- [x] 3.3 Structural invariant tests (single op → binary, ≥2 → fold with ordered ops, `.`/`:` binary incl. flush-as-init interactions, comparison chains still `cclChainedComparisonNode`).
- [x] 3.4 Depth-limit tests updated: chain cases flipped to expect-success (100k-term compile+eval = 100001 committed; statement-mode limit test switched to nesting input); nesting limits still error.
- [x] 3.5 End-to-end `TestDataTable_CCL_LongChainFormula`: 20k-term chains through `AddColUsingCCL` (row-dependent, per-row values verified) and `ExecuteCCL`.
- [x] 3.6 Full `internal/ccl` + root suites green (64 fold subtests).

## 4. Performance gate

- [x] 4.1 Post-change vs baseline (medians of 5): Eval Simple 563→570ns (noise), Eval Medium 1429→1348ns (−5.7%), Eval Chain100 −47%, Eval Chain5000 −60%; Compile Simple 399→409ns (+2.5%, allocs identical 11), Compile Medium +5.8% (+1 net alloc), Compile Chain100 +9% time but −38% allocs. Compile runs once per formula and the delta is repaid within one row of evaluation — gate passed.
- [x] 4.2 One-off end-to-end stress: 4M-term chain now **evaluates correctly** (4000001 in 1.17s); 1.3M nested parens still rejected in 40ms; process alive.

## 5. Docs

- [x] 5.1 `Docs/CCL.md` Limits section updated: chains flattened and uncapped (RAM-bound), nesting limits unchanged, semantics identical.
