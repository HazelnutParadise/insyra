# Proposal: ccl-row-slice-values

## Why

`fix-api-review-batch-8` made `AddColUsingCCL("r", "A:B")` and `LAG(@, 1)` errors. That was the wrong call. Both were producing garbage — a `ccl.ColumnRange` struct in every cell, and the whole flattened table shifted by one — but the garbage was a broken *representation*, not a meaningless *question*.

CCL already resolves a bare column reference against the current row: `Docs/CCL.md:213` says `A.#` is "the value of column A at the current row (same as just `A`)", and `@` on its own has always been the current row. `A:B` is the only column reference that did not follow, and `LAG(@, 1)` — "give me the row before this one" — is exactly what `LAG(@.#, 1)` already did.

So the fix is to complete the resolution rather than reject the expression.

## What Changes

- **A column range on its own is the current row restricted to those columns.** `A:B` gives each cell that row's A and B values (`[]any`), the same as `(A:B).#` already did. It replaces the error batch 8 introduced, and the `ccl.ColumnRange` struct before that.
- **`LAG` and `LEAD` accept a whole row.** `LAG(@, 1)` gives every row the one before it, matching `LAG(@.#, 1)`. `LAG(A:B, 1)` does the same for a subset of columns.
- **Every other sequence function still refuses a row**, with a message saying so: `CUMSUM`, `DIFF`, `PCT_CHANGE`, `CUMPROD`, `CUMMAX`, `CUMMIN` and the `ROLLING_*` family do arithmetic on each element, and a row is never a number.
- **A bare row range stays an error**, because `1:2` says which rows but not of what. The message now says how to write it: `A.(1:2)`.
- The cell holds a slice, not a joined string. `@` and `(A:B).#` already produce `[]any`, so the values keep their types and can be fed to `SUM(...)`; a string is one `&` away when that is what is wanted.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `ccl-type-semantics`: the requirement that a range never becomes a value is replaced by one that says what a range resolves to.

## Impact

- `internal/ccl/ccl_evaluator.go`, `internal/ccl/stdlib_sequences.go`.
- `Docs/CCL.md`, `skills/insyra/references/ccl-operators.md`, both changelogs (amending the batch 8 entry, which is still unreleased).
- Not breaking relative to any released version: `A:B` and `LAG(@, 1)` produced unusable values in v0.3.2 and errors on this branch. Both are now usable.
