# Tasks — bound CCL compile-time recursion

## 1. Parser recursion guard

- [x] 1.1 Add `maxParseDepth = 10000` (const, next to the `parser` struct) and a `depth int` field on `parser`.
- [x] 1.2 In `parsePrimary`: increment `p.depth`, `defer` decrement, and return `fmt.Errorf("expression too deeply nested (max %d levels)", maxParseDepth)` when exceeded.

## 2. AST depth check

- [x] 2.1 Implement iterative `exceedsDepth(n cclNode, limit int) bool` in `ccl_compiler.go` using an explicit stack with early exit; container children per design (binary, chained comparison, func call, assignment, NEW); everything else is a leaf.
- [x] 2.2 Call it via `checkASTDepth` from `CompileExpression` and `compileStatement` after a successful parse; reject with `expression too complex: nesting depth exceeds max %d` (early exit means the exact depth is unknown by design — memory stays O(limit) on deep spines).
- [x] 2.3 Add a comment at the node-type declarations (`ccl_types.go`) reminding that new container node types must be added to `exceedsDepth` and `Bind`.

## 3. Tests

- [x] 3.1 New `internal/ccl/ccl_depth_limits_test.go`: over-cap inputs error promptly (5s guard, reusing the issue-#184 `compileWithTimeout` helper) — nested parens/calls, unary chain, left-spine `+1` chain, statement-mode variant via `CompileMultiline`.
- [x] 3.2 Same file: deep-but-legal inputs still compile — 500-deep nested parens, 500-term `+1` chain, 5000-arg flat call.
- [x] 3.3 End-to-end in `datatable_ccl_test.go`: `AddColUsingCCL`/`ExecuteCCL` with over-cap formulas return promptly, table unchanged, `Err()` set (added to `TestDataTable_CCL_MalformedFormulaReturnsAndSetsErr`).
- [x] 3.4 One-off (not committed): stress-verified the original crash reproducers — 1.3M nested parens returned in 46ms (`expression too deeply nested`), 4M-term `+1` chain in 634ms (`expression too complex`), process alive.

## 4. Docs & follow-up cleanup (same change)

- [x] 4.1 Add a Limits section to `Docs/CCL.md` (both caps = 10000 matching evaluation depth; error messages; `Err()` surfacing; recommendation to cap input length at the application boundary for untrusted input).
- [x] 4.2 Delete the resolved `[2026-07-17]` CCL stack-overflow entry from `AGENTS.md` Follow-ups.

## 5. Verification

- [x] 5.1 `go test ./internal/ccl/ .` full suites green; `go vet` clean; `gofmt -l` clean on touched files.
- [x] 5.2 No measurable compile-time regression for normal formulas (`exceedsDepth` is O(nodes) with early exit, runs once per formula; existing perf-sensitive tests unaffected).
