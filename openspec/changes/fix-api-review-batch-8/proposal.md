# Proposal: fix-api-review-batch-8

## Why

Nine CCL defects share one shape: an expression that cannot mean what it says still produces a value, and the column looks fine. `B != 0 && A / B > 1` reports `division by zero` because `&&` evaluates both sides, so the guard people write to avoid the error is the thing that causes it. `SUM(A B)` — a missing comma — is accepted and sums both columns. `'abc' < 'abd'` and `'abc' > 'abd'` are both false, so strings have no ordering at all, and `'hello' > 5` is false rather than the error the docs promise. `nil & 'x'` writes the Go literal `"<nil>x"` into the cell. `'a' & 1 + 2` fails because `&` binds as tightly as `+`. `A:B` fills every cell with an internal `ccl.ColumnRange` struct the caller cannot even name, and `LAG(@, 1)` fills them with the whole flattened table. `AND()` with no arguments is true, and `AND('abc', true)` is false while `'abc' && true` is an error. `A.(1.7)` reads row 1 and `ROLLING_MEAN(A, 2.9)` uses a window of 2. `A + 0.001` on a date changes nothing, because the fractional hours are truncated away.

Closes #349, #350, #351, #352, #353, #357, #360, #361, #362.

## What Changes

- **`&&`, `||` and `CASE` short-circuit.** The right operand of `&&` is evaluated only when the left is true, `||` only when the left is false, and `CASE` evaluates a branch only when its condition selects it — matching `IF`, `AND()` and `OR()`, which already did.
- **Function arguments must be separated by commas.** A missing comma, a trailing comma or a doubled comma is a compile error.
- **Strings compare lexicographically.** Two operands that are not both numeric compare as text; a string that is not a number compared against a number is an error, as `Docs/CCL.md` has always said. `nil` in a size comparison stays false, also as documented.
- **`&` and `CONCAT` render `nil` as the empty string** instead of Go's `"<nil>"`, matching `UPPER(nil) & 'x'`, which already produced `"x"`.
- **`&` binds looser than `+`/`-`.** `'a' & 1 + 2` is `'a' & 3`, as in Excel. The full precedence table is now in the docs, including the two surprises the review found: `-2^2` is `4` and `2^3^2` is `64`.
- **A range or a whole row never reaches a cell.** An expression whose result is a column or row range is an error naming the operator; sequence functions refuse `@`.
- **`AND()` and `OR()` check their arguments.** They require at least two, and an argument that is not convertible to a boolean is an error rather than a silent `false`.
- **Indices and windows must be whole numbers.** A fractional, NaN or infinite row index, range bound or rolling window is an error instead of being truncated.
- **Adding a fractional number of days to a date keeps the fraction.** `A + 0.001` moves the timestamp by 86.4 seconds.
- **Boolean coercion is documented rather than removed.** `1 && 0` and `'yes' && true` work; the docs claimed they were errors. The coercion table is now written down.

## Capabilities

### New Capabilities

- `ccl-evaluation-order`: a CCL operand is evaluated only when its value can affect the result.
- `ccl-type-semantics`: a CCL operation between values it cannot combine is an error, and no internal type reaches a cell.

### Modified Capabilities

(none)

## Impact

- `internal/ccl/`: `ccl_parser.go`, `ccl_evaluator.go`, `stdlib.go`, `stdlib_sequences.go`; `ccl.go`.
- `Docs/CCL.md`, `skills/insyra/references/`, both changelogs, `api-review.md`.
- **BREAKING.** Four changes alter the value of an expression that previously compiled: `&`'s precedence (`'1' & '2' + 3` was `15`, is now `"15"`), `nil &` (`"<nil>x"` → `"x"`), string ordering (`'abc' < 'abd'` was `false`, is now `true`), and non-numeric mixed comparison (was `false`, now an error). Expressions with a missing comma, a fractional index, a bare range, `LAG(@, …)`, or `AND()` with fewer than two arguments stop compiling or start erroring. Every one of them was producing an answer nobody asked for.
