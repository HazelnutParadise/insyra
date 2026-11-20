# Proposal: py-wide-integers-refused

## Why

An integer that neither an `int64` nor a `uint64` holds came back as the nearest float64 with no error, in an `any`, a table or list cell and every other place, so its last digits were lost without notice (#415). polars 2.0 makes such numbers ordinary: it gives `Int128` for `Int64 + UInt64`, and since `py-results-keep-nulls` a polars DataFrame sends that column's exact Python integers. Measured on 2026-11-20 against `origin/0.4`:

- `2**64 + 4` into `any`, a `[]any`, a `map[string]any`, an `any` field, a table cell or a list cell came back as `1.8446744073709552e+19` with a nil error.
- `-(2**63) - 1` into an `int64` came back as `-9223372036854775808`, a different number, with a nil error.

## What Changes

- The decoder keeps such an integer as its text. Bound into a `float32` or `float64`, it becomes the nearest value, because the result type asked for a float; a `float32` it overflows is an error.
- Bound anywhere else it is an error: `the integer 18446744073709551620 fits neither an int64 nor a uint64; send float(x), or .cast(pl.Float64) for a column, for an approximate number, or str(x), or .cast(pl.String), to keep every digit`.
- An integer too large for a float64, such as `10**400`, still fails while the message is read, so `insyra.Return` still raises, and its error now says to send `str(x)`.
- **BREAKING (behaviour)** on the unreleased 0.4 line. `dev` decodes every number as a float64 by design and is not changed.

## Capabilities

### Modified Capabilities

- `py-run-api`: what an integer beyond 64 bits comes back as.

## Impact

- Code: `restoreNumbers`, `numberTooLarge` and the new `wideInteger` in `py/pyresult.go`; `assignResult`, `assignJSON`, `setInteger`, `holdsWideNumber`, `decodeDataTable` and `decodeDataList` in `py/pyresult_decode.go`.
- Tests: `py/wide_integer_refusal_test.go`, `py/ipc_result_test.go`, the gated end-to-end test in `py/environment_setup_test.go`.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`.
