# Design: parquet-ccl-streamed-sequences

## Context

A sequence function takes whole columns and returns a column of the same length (`internal/ccl/stdlib_sequences.go`). On a loaded table it is evaluated once per expression, and its result is spread over the rows only when it is the whole expression: `CUMSUM(A) + 1` fails with `invalid operands for +`, `CUMSUM(A) == A` is false in every row and `IF(A > 1, LAG(A, 1), 0)` puts the whole sequence in a cell. Inside another sequence function or an aggregate it works, because `evaluateToColumn` passes its column through. `parquet-ccl-whole-file-aggregates` refused every sequence function in `FilterWithCCL` and `ApplyCCL`.

## Decisions

### 1. Only at the top of an expression

A sequence function is streamed when it is the whole filter expression or the whole right-hand side of a `NEW` or an assignment, its column argument holds no sequence function, and its other arguments are constants once whole-file parts are resolved. That covers every placement where the loaded table gives a usable answer except nesting in another sequence function or an aggregate, which would need one stream feeding another; those stay refused and can be added without changing anything here.

### 2. The window functions reuse the function itself

For row `i`, `LAG`, `LEAD`, `DIFF`, `PCT_CHANGE` and `ROLLING_*` read only rows `i - L` to `i + R`, where `L` and `R` follow from the name and the argument (`LAG(x, p)`: `L = p` for `p >= 0`, `R = -p` otherwise; `LEAD` the reverse; `DIFF` and `PCT_CHANGE`: `L = p`; `ROLLING_*(x, w)`: `L = w - 1`). The stream keeps the last `L` inputs before the rows it has not answered, and the `R` inputs whose look-ahead has not arrived. Each push calls the registered function on that kept history plus the new values and emits the outputs whose window it fully held. At the end of the file it calls the function on what is left and emits the rest, so a shift past the end gives `nil` exactly as on the whole column. Because the function itself computes every output, from the same values in the same order, the result is bit for bit the whole-column one; the only thing the stream decides is which outputs are complete.

### 3. The cumulative functions share their accumulator

Prepending the last running value to the next batch would be shorter, but it is wrong: a running sum that reached `NaN` through `+Inf` and `-Inf` would be read back as a missing value and restart at 0. The loop of `seqCumImpl` moves into a small accumulator type that both the function and the stream use, so the stream carries the exact state, `NaN` included.

### 4. Batches become runs of columns

`LEAD` cannot answer the last `R` rows of a batch until the next batch arrives. In `ApplyCCL` the statement after it may read the column it writes, so those rows cannot go on to the next statement either. `ApplyCCL` therefore runs as a pipeline: each statement is a stage that takes runs of consecutive rows and passes on the rows it has finished, in order; a stage without a sequence function finishes every row at once. The runs are held as columns of Go values, the form `applyStatements` already built for every batch, instead of an Arrow record with overrides for the columns statements wrote. The writer turns each run that leaves the last stage into a record, so the output's row groups still follow the file's layout. `FilterWithCCL` keeps the rows of the batches a sequence filter has not answered and decides them when the values arrive. When resolving statement `k`, the batches handed to `ResolveWholeTable` are the runs leaving stage `k - 1` of a fresh pipeline.

### 5. Registered functions

A sequence function registered by a caller, or a built-in name a caller re-registered, has no streaming form, like a re-registered aggregate. The third change computes those.

### 6. The type of a written column

A sequence function's first rows are `nil`, so a created column's first batch can hold nothing else, and `ApplyCCL` used to settle each new column's type from its first batch and write `nil` as 0. Knowing a column's type before writing needs every value, which streaming does not have. `ApplyCCL` therefore guesses from the first rows it writes and checks every later run against the guess; when a later value would change a type, it abandons the temporary file, reads the file once more to settle every written column's type from all its values, and writes again. The common script costs no extra read. A column the file had keeps its own type when every value fits it without loss, because CCL computes every number as `float64` and widening `['n'] = A * 2` would change the file's schema for nothing; a fraction does not fit an integer column and widens it, where it used to be cut off. Every written column is nullable.

## Risks

- `LEAD(x, n)` holds `n` rows of every column, and `ROLLING_*(x, w)` keeps `w - 1` values of `x`; a shift or window as long as the file holds the file. That is inherent in the function, not in batching.
- Converting every batch to Go values costs what `ApplyCCL` already paid; `FilterWithCCL` used to read the current row's cells through the record and now reads them from columns built once per batch.
