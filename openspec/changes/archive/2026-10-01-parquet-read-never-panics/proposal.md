# Proposal: parquet-read-never-panics

## Why

A Parquet file whose data pages and footer disagree made the Arrow reader dereference a nil pointer, and the panic reached the caller, against this line's rule that the library never panics. Measured on 2026-09-30: laying the first k bytes of a 1,000-row file over a 3,000-row file, for k from 100 upward in steps of 53, made `ReadFrom` panic in 169 of 655 cases. Two concurrent `Write` calls sharing the old fixed `<path>.tmp` produced exactly such files until `parquet-write-options`, but a file damaged anywhere else reaches the same code. The owner approved the fix on 2026-10-01, the `AGENTS.md` follow-up recorded on 2026-09-30.

## What Changes

- `Read`, `ReadFrom`, `ReadColumn` and `Inspect` turn a panic inside the Arrow reader into an error naming the file, `parquet: <path> is not a readable Parquet file: <cause>`.
- `Stream`, `StreamFrom`, `FilterWithCCL` and `ApplyCCL` do the same inside the goroutine that reads the file, so the error arrives where a read error does and nothing ends the program. `ApplyCCL` leaves the original file as it was.

## Capabilities

### New Capabilities

- `parquet-damaged-files`: what the Parquet readers do with a file they cannot make sense of.

### Modified Capabilities

None.

## Impact

- `parquet/api.go`, `parquet/internal.go`, a new `parquet/read_damaged_test.go`.
- `Docs/parquet.md`, both changelogs, `AGENTS.md` (the follow-up is resolved and removed), `delivery-status.md`.
