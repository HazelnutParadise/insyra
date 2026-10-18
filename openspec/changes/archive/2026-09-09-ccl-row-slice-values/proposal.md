# Proposal: ccl-row-slice-values

## Why

A column range used as a whole expression, `AddColUsingCCL("r", "A:B")`, wrote an internal `ColumnRange` struct into every cell, and `LAG(@, 1)` shifted the whole flattened table into one nonsense slice repeated in every row. Both were a broken representation of a sensible question. CCL already resolves a bare column reference against the current row (`A` means `A.#`, `@` means `@.#`), and `A:B` was the only column reference that did not follow. The change is ported from `0.4`, where it revised an earlier decision to refuse both forms.

## What Changes

- A column range on its own is the current row restricted to those columns: `A:B` gives each cell that row's A and B values as a `[]any`, the same as `(A:B).#`. It no longer produces a `ColumnRange`.
- `LAG` and `LEAD` accept a whole row: `LAG(@, 1)` gives every row the one before it, matching `LAG(@.#, 1)`; `LAG(A:B, 1)` does the same for a subset of columns.
- Every other sequence function refuses a row with a message saying so, because they do arithmetic on each element.
- A bare row range such as `1:2` is an error that says how to write it: `A.(1:2)`. It used to put a `RowRange` into the cells.
- Ranges inside an aggregate and row ranges attached to a column read exactly as before (`SUM(A:C)`, `SUM(A.(0:1))`).

## Capabilities

### Modified Capabilities

- `ccl-type-semantics`: what a range and `@` resolve to when they are a whole expression.

## Impact

- `internal/ccl/ccl_evaluator.go`, `internal/ccl/row_slice.go` (new), `Docs/CCL.md`, both changelogs. The previous results were an unusable internal struct and a nonsense shifted slice, so no usable result changes.
