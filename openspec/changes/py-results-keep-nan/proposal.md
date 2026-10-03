# Proposal: py-results-keep-nan

## Why

A value Python passes to `insyra.Return` that holds a NaN or an infinity came back as `nil` with no error. Python's `json.dumps` writes those floats as `NaN`, `Infinity` and `-Infinity`, which are not JSON, so go-json refused the message: `invalid character 'N' looking for beginning of value`. The IPC handler logged that and closed the connection without an acknowledgement. `insyra.Return` read the closed connection as the end of a normal exchange, marked its result sent and exited 0, and the runner, finding no result, returned `nil` and a nil error. The adversarial review of `py-ipc-server-errors` measured this on 2026-09-30 with the pinned environment: `RunCode(&dt, "insyra.Return(pd.DataFrame({'a': [1.0, float('nan')]}))")` returned a nil `*DataTable`. A DataFrame with a missing value is ordinary data.

The same path loses every other message go-json refuses. Measured on 2026-10-03, an integer too large for a float64, such as `10**400`, fails with `strconv.ParseFloat: … value out of range` and would come back as `nil` the same way.

## What Changes

- A NaN, `inf` or `-inf` in a result comes back as `math.NaN()`, `math.Inf(1)` or `math.Inf(-1)`: in a table or list cell, a float field, slice, array or map value, or anything decoded into `any`. insyra holds `nil` and `NaN` as two values, each with its own methods (`ClearNils`, `ClearNaNs`, `ReplaceNaNsWith`), so `None` still comes back as `nil` and a NaN as a NaN. A NaN into a type that cannot hold one, such as `int`, is an error.
- The Go side reads Python's `NaN`, `Infinity` and `-Infinity` when go-json refuses a message: it marks them with numbers Python never writes, decodes with numbers kept as text, and turns the marks back into float64 values. Messages without them decode exactly as before.
- A message the Go side cannot read is answered with an error, and the call returns it: the reason is kept for the run whose ID the message starts with, and becomes the call's error when the run ends without delivering another result, since Python code can catch the exception and return something else. `insyra.Return` raises when the answer is an error or when no answer comes, instead of reporting success.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: how a result holding a NaN or an infinity is decoded, and what a result the Go side cannot read does.

## Impact

- Code: `py/pyresult.go` (the IPC handler and the message decoding), `py/pyresult_decode.go` (binding a NaN or an infinity outside tables), and `insyra.Return` in `py/builtin.go`.
- Tests: `py/ipc_result_test.go`, `py/nonfinite_bind_test.go`, `py/nonfinite_run_test.go`, the fake Python in `py/main_test.go`, and the gated end-to-end test.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, and `AGENTS.md`, where the follow-up is deleted.
