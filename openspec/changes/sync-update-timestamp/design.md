# Design — synchronous updateTimestamp

## Decision 1 — plain synchronous call, not coalescing

The Follow-ups entry offered two options: synchronous update or coalescing (batching timestamp bumps). Measurement settles it: the synchronous call is already ~10 ns of atomics on a path that costs ≥88 ns total, so there is nothing left worth coalescing. Coalescing (a dirty flag + flusher, or rate-limited stores) would add state and ordering questions to save single-digit nanoseconds. Rejected.

## Decision 2 — mechanical substitution only, no signature or body changes

`updateTimestamp` bodies stay exactly as they are (monotonic guard included — it now also protects against clock regressions within the same second, same as before). The edit is `go <recv>.updateTimestamp()` → `<recv>.updateTimestamp()` at all 102 sites; receivers are `dl`, `dt`, `col`, and 4 indexed `dt.columns[colNo]` forms (the indexed forms are why simple-identifier greps count only 98 — the post-edit zero-remaining check is the authoritative guard). No site does anything else in the spawned statement.

## Decision 3 — deadlock safety argument

All call sites run inside `AtomicDo`-serialized method bodies. `updateTimestamp` takes no lock (pure `atomic.Int64` load/store), so inlining it into the locked section cannot re-enter the actor and cannot deadlock. This was the "first confirm" condition on the Follow-ups entry; confirmed by reading both implementations (`datalist.go`, `datatable.go`) before measuring.

## Decision 4 — pin the new contract with an in-package test

The only externally observable change is positive: `GetLastModifiedTimestamp()` is correct the moment a mutating call returns. Second-granularity makes a black-box test flaky, so the regression test is white-box (package `insyra`): store 0 into `lastModifiedTimestamp`, mutate, assert non-zero immediately. Under the goroutine version this has a genuine scheduling race and would flake toward failure; under the synchronous version it is deterministic. One test per type (DataList, DataTable) plus one column-receiver path (`col.updateTimestamp` via a DataTable column mutation).

## Failure modes considered

- **A site whose `go` statement wraps more than the timestamp call**: none exist (Decision 2 enumeration); the sed pattern requires the exact one-line form, so any such site would simply be left untouched and caught by the post-edit `grep -c "go .*updateTimestamp"` == 0 check.
- **Tests that depended on the async window**: existing timestamp tests (`TestDataListGetLastModifiedTimestamp`, `TestDataTable_GetLastModifiedTimestamp`, `TestErrorTimestamp`) pass under the synchronous variant (verified during the A/B run) — they sleep ≥1 s, so they never observed the async gap.
- **Longer actor-lock hold time**: +2 atomic ops + one `time.Now()` (~tens of ns) per mutation inside the serialized section; dwarfed by the 200+ ns saved by not spawning. Net contention strictly decreases (parallel benchmark: 627 → 146 ns/op).
