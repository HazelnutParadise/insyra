# Proposal: core-lock-and-snapshot

## Why

Four places in the core promise protection they do not give.

`AtomicDoAll(f func(), instances ...any)` accepts anything. A value it cannot lock gets a warning in the log, and then the callback runs without that value locked, so a caller who passed the wrong thing believes they are protected and is not. A nil `*DataList` is dereferenced and panics.

`GroupBy` copies the parent's column *pointers*. The groups are decided from the rows as they are when `GroupBy` runs, but `Aggregate` reads the parent's data when it runs, so a change in between reaches the result — a group that held `3` sums to `102` — and `Aggregate` reads that data without holding any lock. The race detector confirms it.

`ExecuteCCL` applies each statement to the table as it goes. When statement *n* fails, statements 1 to *n*-1 stay applied, and nothing tells the caller the table is half-changed.

A custom CCL function that reads another table races with a writer of that table. Nothing documents it, and the fix the review suggested — take the other table's lock from inside the function — trades the race for a deadlock when two goroutines do the mirror image. Both were measured.

Separately, several tests change the global `Config` and do not put it back, so the tests after them run under settings they did not choose.

Closes #209, #229, #234, #310, #363.

## What Changes

- **`AtomicDoAll` takes `...Lockable`.** `*DataList`, `*DataTable` and the `isr` wrappers, which embed them, satisfy it; anything else no longer compiles. A nil instance is skipped.
- **`GroupBy` copies the column data it groups**, so `Aggregate` works on the table the groups were built from and never reads the parent without its lock.
- **`ExecuteCCL` is all or nothing.** The script runs against a private working copy and is written back only when every statement succeeds. Existing columns keep their `DataList` objects and take the new data.
- **`Docs/CCL.md` documents custom functions**, which it did not mention at all, including the one locking pattern that is both race-free and deadlock-free: lock every table involved with `AtomicDoAll` from outside the CCL call.
- **Tests restore the configuration they change.** The root package sets its baseline once in a `TestMain` instead of two `init()` functions; `restoreConfig` also covers coloured output and the error hook; other packages restore the log level through `t.Cleanup`.

## Capabilities

### New Capabilities

- `core-lock-and-snapshot`: a locking or snapshotting call either protects what it says it protects or refuses to compile.

### Modified Capabilities

(none)

## Impact

- `atomic.go`, `datatable.go`, `datatable_groupby.go`, `datatable_ccl.go`; tests across the root package, `isr`, `stats`, `mkt`, `ml`, `ml/mltest`.
- `Docs/CCL.md`, `Docs/DataTable.md`, `Docs/DataList.md`, `skills/insyra/SKILL.md`, both changelogs, `api-review.md`.
- **BREAKING.** `AtomicDoAll` called with a `[]any` slice or an `any` value no longer compiles; change the slice to `[]insyra.Lockable`. A failed `ExecuteCCL` leaves the table unchanged instead of half-changed.
- `ExecuteCCL` still copies the table once per statement so each statement sees its predecessors; that cost is unchanged and remains in CCL-15.
- No `ExecuteCCLErr`: the typed CCL errors already reach callers through `errors.As(dt.Err(), …)`, so a second entry point would carry the same information.
