# Proposal: ccl-error-reporting

## Why

When a CCL expression fails, the message does not say enough to fix it, and in two places it says the wrong thing.

A failure looks like `Failed to apply CCL on DataTable after 213.792µs: division by zero`. That prefix is identical whether the expression would not compile or blew up on row 40,000, so the first question — is my formula malformed, or is my data? — is not answered. When it is the data, the row is not named, so on a large table there is nothing to look at.

`position N` means two different things: in the tokenizer it is a byte offset into the expression, in the parser it is an index into the token list. `SUM(A B)` reports "position 3" meaning the fourth token, which points at nothing a reader can find in the text they wrote.

Two messages print Go internals: `unexpected token: {5 )}` is a `cclToken` struct rendered with `%v`, and `invalid range operands: &{: 0x1d48…}` is an AST node with a pointer in it.

And every error is a bare `fmt.Errorf` string, so a caller who wants to react differently to a typo than to a bad row has to match on text.

Closes #354.

## What Changes

- **Two error types, exported.** `ccl.CompileError` carries the expression, the byte offset, the source text at that offset and the message; `ccl.EvalError` carries the expression, the row and the cause it wraps. Both are re-exported from `engine/ccl`, and `EvalError` unwraps.
- **`errors.As` reaches them through `Err()`.** `ErrorInfo` gains a `Cause` field and an `Unwrap`, so `errors.As(dt.Err(), &compileErr)` works after `AddColUsingCCL`.
- **One meaning for a position.** Every token records its byte offset in the expression, and every compile error reports that offset plus the text at it. The parser stops reporting token indices.
- **No Go internals in a message.** The two `%v`-on-a-struct sites print the token's text and a description of the operands instead.
- **A runtime failure names its row**, or says the expression does not depend on the row when it does not.
- The elapsed time is dropped from the failure message. It was only ever noise on a failure; it stays in the debug log on success.

## Capabilities

### New Capabilities

- `ccl-error-reporting`: a failed CCL expression says which phase failed, where in the expression or which row, in terms the caller can act on and match on.

### Modified Capabilities

(none)

## Impact

- `internal/ccl/`: new `errors.go`; `ccl_tokenizer.go`, `ccl_parser.go`, `ccl_compiler.go`, `ccl_evaluator.go`.
- `engine/ccl/ccl.go`, `error_buffer.go`, `datatable.go`, `ccl.go`, `datatable_ccl.go`.
- `Docs/CCL.md` (troubleshooting), both changelogs.
- Message text changes; nothing that was working stops working.
