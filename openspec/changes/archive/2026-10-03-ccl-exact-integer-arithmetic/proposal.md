# Proposal: ccl-exact-integer-arithmetic

## Why

Every number CCL computed went through `float64`. An integer column came out of any arithmetic, even `A * 1`, as a `float64` column, and an integer past 2^53 lost its last digits with no error: measured on 2026-10-03, `int64(9007199254740993) + 0` gave `9007199254740992`, and `SUM` over such a column was off the same way. IDs, order numbers and counts are where this bites. pandas, polars and SQL all keep `int64 + int64` an integer. Issue #358 (CCL-21). The documentation half shipped in ad10525e; on 2026-10-04 the owner decided `0.4` keeps integers exact.

## What Changes

- **BREAKING**: a number literal written as digits alone (`1`, `-7`, `9007199254740993`) is an `int64`. A literal with a decimal point or an exponent, or past the `int64` range, is a `float64` as before. Without this, `A + 1` would still leave an integer column through `float64`.
- **BREAKING**: when both operands are integers (any Go integer type, a `uint64` only up to the `int64` maximum), `+`, `-`, `*`, `%` and unary minus compute in `int64` and give an `int64`. `==`, `!=`, `<`, `>`, `<=` and `>=` compare two integers exactly. In arithmetic a `nil` next to an integer is an integer `0`, so an integer column with missing cells stays an integer column through `A + 1`.
- An integer result `int64` cannot hold is an error naming the operation, never a wrapped number or a `float64` that lost digits.
- **BREAKING**: `SUM`, `MIN` and `MAX`, and their streaming forms used by `parquet.FilterWithCCL` and `ApplyCCL`, give an `int64` when every value they use is an integer; an integer `SUM` past `int64` is an error. `MOD` follows `%`.
- **BREAKING**: the row index `#` is an `int64`.
- `TOSTR`/`TEXT` with a float verb formats an integer as the `float64` it equals, so `TOSTR(50, '%.1f')` is `"50.0"`; it also makes `TOSTR(A, '%.2f')` work on an integer column, which used to fail.
- Unchanged: `/` and `^` always give a `float64`, as do `AVG`, `MEDIAN`, the variance family, `COUNT`, the other math functions, the sequence functions, and any operation with a `float64`, a numeric string or a boolean on either side. `nil + nil` stays `float64` 0.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ccl-type-semantics`: integers stay integers; the `uint64` scenario of "Every Go numeric type is a number" now keeps every digit.

## Impact

- `internal/ccl`: `integer_arith.go` (new: `exactInt`, the int64 operators with overflow checks, `integerAggregate`), `ccl_parser.go` (integer literals), `ccl_types.go` (`cclIntegerNode`), `ccl_evaluator.go` (the integer path in `applyOperator`, `#`, folded literals), `stdlib.go` (`SUM`, `MIN`, `MAX`), `stdlib_math.go` (`MOD`), `stdlib_typeconv.go` (`TOSTR`), `stream_aggregates.go`.
- Callers that type-assert an integer result as `float64` break. A column filled from a literal fallback, such as `COALESCE(TONUM(A), 0)`, holds integer `0`s next to `float64` values; `0.0` keeps it `float64`.
- `parquet.ApplyCCL`: a column whose new values are all integers and do not fit its file type widens to `int64` rather than `float64`.
- Tests that pinned `float64` results of integer arithmetic move to `int64`; no value changes.
- `Docs/CCL.md`, `Docs/parquet.md`, `skills/insyra/SKILL.md`, both changelogs, `api-review.md`, `delivery-status.md`.
