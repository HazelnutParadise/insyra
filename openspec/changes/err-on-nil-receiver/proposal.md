## Why

`Err()` on a nil `*DataList` or `*DataTable` dereferenced the receiver and crashed, and a nil is exactly what a lookup such as `GetCol` returns when it finds nothing. The library's rule is that it never terminates by default, yet the method a caller uses to ask what went wrong was the one that brought the program down. The AGENTS.md follow-up of 2026-09-10 proposed guarding `Err()` alone; the owner ruled on 2026-09-26 to do exactly that.

## What Changes

- `Err()` on a nil `DataList` or `DataTable` returns a new `*ErrorInfo` saying `nil DataList` / `nil DataTable`.
- No other method is guarded: a nil value still fails loudly everywhere else, so it cannot travel on as if it were an empty value.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `instance-error-contract`: `Err()` answers on a nil receiver.

## Impact

- `datalist.go`, `datatable.go`, `error_buffer.go`, `err_nil_receiver_test.go`; `Docs/DataList.md`, `Docs/DataTable.md`, both changelogs; the AGENTS.md follow-up is removed.
