# Proposal: ccl-doubled-quote-escape

## Why

A CCL string literal had no escape, so a string holding its own quote character could not be written, and one holding both kinds of quote could not be written at all: `'it\'s'` failed with `unclosed string`. `Docs/CCL.md` also said strings use single quotes, although double quotes work the same way. Issue #364 (CCL-27).

## What Changes

- Inside a quoted literal, the quote character written twice stands for one, as in Excel formulas: `'it''s'` is `it's` and `"say ""hi"""` is `say "hi"`. The owner chose Excel's form over a backslash escape for this backlog: CCL is Excel-like, and a backslash escape would change what an existing literal such as `'C:\data'` means.
- No expression that compiled changes meaning. Checked on 2026-10-03: two quotes in a row always ended one literal and began another, and the parser has no place where a string may follow a string, so `'it''s'`, `''''` and `CONCAT('a''b', 'c')` all failed with `unexpected token` or `expected ',' or ')'`. `['it''s']` failed with `expected ']' after column name reference`.
- A bracketed column name follows the same rule: `['O''Brien']` is the column `O'Brien`.
- `CompileMultiline` splits a script into statements consistently with the tokenizer. Its splitter already sees a doubled quote as a quote that closes the literal and opens it again at once, which leaves `;` and line breaks inside the literal alone; a test now pins that.
- A backslash stays an ordinary character.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: a doubled quote inside a literal.

## Impact

- `internal/ccl/ccl_tokenizer.go`: string literals and bracketed column names read through one helper, `scanQuoted`. `internal/ccl/ccl_compiler.go`: a comment on the splitter.
- Not breaking: only inputs that failed to compile now compile.
- `Docs/CCL.md`, both changelogs, `api-review.md`, `delivery-status.md`.
