# Bound CCL compile-time recursion

## Why

While verifying issue #184 (tokenizer hang, fixed in 58a9385), an adversarial audit found that the CCL *compile* pipeline has no recursion bound. `maxEvalDepth` (10000) protects only the evaluator; three compile-time walkers recurse unbounded:

1. `parseExpression` ↔ `parsePrimary` mutual recursion (`internal/ccl/ccl_parser.go`) — deep nesting such as `((((...` recurses once per level.
2. `Bind` (`internal/ccl/ccl_compiler.go`) — recurses once per AST level; a left-associative chain `1+1+1+...` parses with O(1) recursion (iterative precedence loop) but produces an O(n)-deep AST.
3. `IsRowDependent` / `containsRowIndex` (`internal/ccl/ccl_evaluator.go`) — same AST-depth recursion, also called before evaluation.

Go stack overflow is a **fatal runtime error, not a recoverable panic** — the `recover()` in the CCL entry points cannot catch it. One adversarial formula string therefore kills the entire host process. Measured on go1.25.12 windows/amd64 (1GB default stack): ~1.3M nested parens crash the parser; ~4M `+1` terms crash `Bind`. Reachable from every public CCL entry (`AddColUsingCCL`, `EditColByIndexUsingCCL`, `EditColByNameUsingCCL`, `ExecuteCCL`), i.e. from any service that lets end users type CCL.

## What Changes

- Add a parse-recursion depth guard: a `depth` counter on the `parser` struct, incremented in `parsePrimary`, capped at `maxParseDepth` (10000). Exceeding it returns an `expression too deeply nested` error instead of recursing toward stack exhaustion.
- Add a post-parse AST depth check in `CompileExpression` and `compileStatement`: an **iterative** (explicit-stack) `astDepth` walker rejects ASTs deeper than `maxEvalDepth` with an `expression too complex` error. This closes the left-spine vector that the parse guard cannot see, and protects `Bind`/`IsRowDependent`/`Evaluate` at their single upstream chokepoint (AST node types are unexported, so the parser is the only AST producer).
- Regression tests at both layers: compile-level (over-cap nested parens, unary chains, left-spine `+1` chains must error promptly; representative deep-but-legal inputs must still compile) and end-to-end (`AddColUsingCCL` with an over-cap formula returns, leaves the table unchanged, sets `Err()`).
- Docs: add a Limits note to `Docs/CCL.md` (nesting/complexity caps and the errors they produce).
- Delete the now-resolved `[2026-07-17]` Follow-ups entry from `AGENTS.md`.

## Capabilities

### New Capabilities
- `ccl-compile-limits`: CCL compilation rejects over-deep or over-complex expressions with a prompt, descriptive error instead of exhausting the process stack.

### Modified Capabilities
<!-- None. No existing CCL capability spec; behavior of all evaluable expressions is unchanged. -->

## Impact

- **Code**: `internal/ccl/ccl_parser.go` (depth guard), `internal/ccl/ccl_compiler.go` (`astDepth` + checks), new `internal/ccl/ccl_depth_limits_test.go`, additions to `datatable_ccl_test.go`.
- **Docs**: `Docs/CCL.md` Limits note; `AGENTS.md` follow-up entry removal. `skills/insyra/` needs no update (its references do not document engine limits or error strings).
- **Behavior**: only pathological inputs are affected. Any expression deeper than `maxEvalDepth` already failed at evaluation ("maximum recursion depth exceeded") or crashed the process, so no previously *usable* expression is rejected. Errors surface through the existing warn-and-`Err()` path; no API change, no new dependencies.
- **Out of scope**: transient memory proportional to input size (tokens + AST are allocated before rejection). That is linear resource use, not a crash vector; callers exposing CCL to untrusted users should still cap input length at their boundary.
