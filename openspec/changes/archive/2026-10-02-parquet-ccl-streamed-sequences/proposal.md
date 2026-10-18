# Proposal: parquet-ccl-streamed-sequences

## Why

`FilterWithCCL` and `ApplyCCL` read a Parquet file 1,000 rows at a time. `parquet-ccl-whole-file-aggregates` made aggregates, the row index and fixed rows give the answer the loaded table gives, and refused what it could not compute that way, sequence functions among them: before it, `ApplyCCL(ctx, path, "NEW('c') = CUMSUM(A)")` wrote each batch's whole sequence, as text, into every cell, and since it the call is an error. On 2026-10-01 the owner chose to compute these answers without holding whole columns, in three changes; this is the second, for the built-in sequence functions.

## What Changes

- A built-in sequence function (`LAG`, `LEAD`, `DIFF`, `PCT_CHANGE`, `CUMSUM`, `CUMPROD`, `CUMMAX`, `CUMMIN`, `ROLLING_SUM`, `ROLLING_MEAN`, `ROLLING_MIN`, `ROLLING_MAX`, `ROLLING_STD`) written as a whole filter expression, or as the whole right-hand side of a `NEW` or an assignment in `ApplyCCL`, is computed over the whole file and gives what the loaded table gives, value for value. Memory holds the rows the function's window or shift reaches across a batch boundary, not the column: the last `n` values for `LAG(x, n)`, `DIFF` and `PCT_CHANGE`, the last `w - 1` for `ROLLING_*(x, w)`, the running value for the cumulative functions, and the next `n` rows of every column for `LEAD`, whose rows are held back until those arrive.
- The file is read once for them; no extra pass.
- Still refused, with an error naming it: a sequence function inside another expression, inside another sequence function or inside an aggregate (`CUMSUM(A) + 1`, `LAG(CUMSUM(A), 1)`, `SUM(CUMSUM(A))`), one a caller registered or re-registered under a built-in name, one whose column argument does not change from row to row, and one whose shift or window does. On the loaded table the first group fails or gives a slice per cell, so refusing them loses no answer anyone could use.
- `ApplyCCL` writes a column a statement assigns with the type it had when every new value fits it without loss, and otherwise, like a column the script creates, with the type `Write` gives those values, settled from every value rather than the first 1,000 rows; every written column is nullable. A sequence function answers `nil` for its first rows, which a created column wrote as 0 and settled as text when its first batch held nothing else, and a fraction assigned to an integer column was truncated. A value or column type it cannot write is an error, not a panic.
- `ApplyCCL` runs its statements as a pipeline of stages over runs of rows, so a statement that holds rows back, `LEAD`, passes the rest on in order and the statements after it see every column the statements before it wrote.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `parquet-ccl-batches`: the built-in sequence functions at the top of an expression are computed over the whole file; the rest of what is refused stays refused.
- `parquet-ccl-output`: the type of every column a statement writes loses no value.

## Impact

- `internal/ccl`: a streaming form of each built-in sequence function, sharing its arithmetic with the function itself, and the check that a sequence function stands where it can be streamed.
- `parquet/ccl.go`: batch contexts held as columns instead of an Arrow record with overrides, `ApplyCCL` as a pipeline of statement stages, `FilterWithCCL` deciding rows a sequence holds back once their values arrive.
- `Docs/parquet.md`, both changelogs, `AGENTS.md` (the follow-up narrows to the third change), `delivery-status.md`.
