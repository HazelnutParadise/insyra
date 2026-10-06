# Proposal: engine-ccl-opaque-types

## Why

`engine/ccl` lets a program apply CCL to its own data. Since it was added (19f49d7a, 2026-01-18) it has published the internal AST through four type aliases, which #259 (EN-1) flagged. `engine-ccl-registry-contract` fixed the rest of #259; the aliases waited for the owner, who decided on 2026-10-06. Measured on that day:

- `CCLNode` is an alias of `any`, so `ccl.Evaluate("A + B", ctx)` and `ccl.Evaluate(42, ctx)` compile, and fail only when run, with `invalid node`.
- `MapContext` exports `Data`, `Rows`, `CurrentRowIdx`, `ColNameMap` and `ColNames`. `ctx.CurrentRowIdx = 99` makes `A` evaluate to `nil` with no error, where `SetRowIndex(99)` refuses with `row index 99 out of range`. A column added through `ctx.Data["C"]` always reads as `nil`, because the name-to-column map is not updated.
- `Context` is the interface a program implements, so adding a method to it breaks every implementation outside the module, and nothing said it would not change.
- `EvaluationResult` is a plain result struct.

## What Changes

- `CCLNode` becomes `struct{ node internalccl.CCLNode }`. `CompileExpression`, `CompileMultiline`, `CompileMultilineStatements`, `Bind`, `GetNewColInfo` and `GetExpressionNode` wrap what the compiler returns, and the functions that take a node unwrap it. `Evaluate`, `EvaluateStatement` and `Bind` refuse the zero `CCLNode{}` with an error. `CompiledStatement` becomes a struct of its own with the same `Node` and `Src` fields. **BREAKING**: passing anything but a node no longer compiles, and a node is compared with `ccl.CCLNode{}` instead of `nil`.
- `MapContext` becomes a wrapper with no exported fields, made by `NewMapContext`, implementing every `Context` method by delegation. The zero `MapContext` refuses reads with an error instead of panicking. **BREAKING** for code that read or set the fields.
- `Context` stays an alias, and its doc says the method set is fixed: a capability CCL gains later comes as an optional interface the evaluator checks for, as `GlobalRowContext` already does. A test pins the fifteen methods, and the internal interface carries a maintainer note.
- `EvaluationResult` stays an alias, documented.
- The internal packages do not change, apart from that note, so the root package, `parquet` and the CLI are untouched.

## Capabilities

### New Capabilities

- `engine-ccl-api`: what `engine/ccl` promises a program that compiles and evaluates CCL against its own data.

### Modified Capabilities

(none)

## Impact

- Code: `engine/ccl/ccl.go`, new `engine/ccl/map_context.go`, the doc of `Context` in `internal/ccl/context.go`.
- Tests: new `engine/ccl/opaque_types_test.go`; three `nil` comparisons in `engine/ccl/ccl_test.go`.
- Docs: `engine/README.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
