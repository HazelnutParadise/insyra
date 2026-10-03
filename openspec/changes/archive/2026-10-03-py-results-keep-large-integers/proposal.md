# Proposal: py-results-keep-large-integers

## Why

An integer above 2^53 in the value Python passes to `insyra.Return` comes back with its last digits changed. The IPC handler decodes every JSON number into a float64 before the result reaches the type it is bound into, and a float64 does not hold every integer past 2^53. Measured by the review of `py-nested-table-results` on 2026-10-01 and again on 2026-10-03: `9007199254740993` comes back as `9007199254740992`, into an `int64` field too. A database id or a nanosecond timestamp returned from Python is that large, and the wrong value arrives with no error.

## What Changes

- The handler keeps an integer exact when it decodes the message. A message holding sixteen or more digits in a row that are not part of a decimal, which an integer above 2^53 needs, is decoded with its numbers kept as text; any other message decodes as before, since every number in it is exact as a float64. Digits of a decimal do not count, because Python prints most floats with sixteen or seventeen of them, and counting them sent a 19 MB message of floats the slower way, 330 ms against 113 ms.
- An integer above 2^53 in magnitude comes back as an `int64`, or a `uint64` above the `int64` range, in a table or list cell, an `any`, or anywhere JSON would have put a float64. Every other number keeps the type it had: an integer within ±2^53, which a float64 holds exactly, is still a float64, so no value that came back right changes type. Python writes an int without a decimal point or exponent and a float always with one, so a float is never read as an int.
- Bound into an integer field such as `int64` or `uint64`, an integer is exact at any size the field holds.
- A type that holds an `any` is decoded part by part, the way a type holding a table already is, so an `any` inside a struct, map or slice keeps a large integer too.
- An integer beyond 64 bits still comes back as the nearest float64, and one beyond the float64 range is still an error.
- `insyra.Return` converts each column of a pandas or polars DataFrame on its own. Measured on 2026-10-03 with pandas 3.0.6 and polars 1.44.2, `to_numpy()` on a whole frame turned an `id` column beside a float column into floats before sending, so `2**53 + 1` left Python as `9007199254740992.0`; this is the most common shape of a DataFrame holding ids. A frame of one type is sent exactly as before.
- An integer field takes a number only when its type holds it. go-json v0.10.6 wraps a nineteen- or twenty-digit number around in an `int64`, `int` or `uint64` with no error, so a result holding a number that can do so is decoded part by part, where an integer field is set with a range check.
- An `any` that already holds a non-nil pointer is decoded into what the pointer points to, as `encoding/json` does; the adversarial review of the first version found it replaced instead, once types holding an `any` were decoded part by part.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: how an integer in a result is decoded.

## Impact

- Code: `decodeResultMessage` and `restoreNumbers` in `py/pyresult.go`, and the predicate in `py/pyresult_decode.go` that sends a type to the part-by-part decoder.
- Tests: `py/ipc_result_test.go`, `py/large_integer_test.go`, and the gated end-to-end test.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, and `AGENTS.md`, where the follow-up keeps only the part about key order it also recorded.
