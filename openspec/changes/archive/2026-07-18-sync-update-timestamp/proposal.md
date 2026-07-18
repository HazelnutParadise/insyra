# Update lastModifiedTimestamp synchronously

## Why

Every mutating method on `DataList`/`DataTable` ends with `go x.updateTimestamp()` — 102 sites (39× `dl`, 47× `dt`, 12× `col`, 4× indexed `dt.columns[colNo]`) across 10 files. The spawned goroutine's entire job is `time.Now().Unix()` plus an atomic load/store (tens of ns), while the `go` statement itself costs an order of magnitude more and allocates. Under tight-loop mutation this floods the scheduler with short-lived goroutines (the reason the `-race` stress tests had to be bounded). Tracked as the `[2026-07-10]` Follow-ups entry in `AGENTS.md`.

A/B measurement on the real library (i7-13700, go1.25.12 windows/amd64, 5-run medians, identical code except the `go` keyword):

| Scenario | `go` (current) | sync | delta |
|---|---|---|---|
| `DataList.Append` serial | 302 ns/op, 1 alloc/op | 88 ns/op, 0 allocs/op | 3.4× faster |
| `Append` parallel (own instance each) | 627 ns/op | 146 ns/op | 4.3× faster |
| `DataTable.AppendRowsByColIndex` | 559 ns/op | 301 ns/op | 1.9× faster |
| 1M-append flood: loop time / peak goroutines / quiesce | 310 ms / ~85 / ~1 ms | 92 ms / 3 / 0 | 3.4× faster, flood gone |

Safety precondition (per the Follow-ups entry) confirmed: both `updateTimestamp` implementations are pure atomics (`lastModifiedTimestamp.Load/Store` with a monotonic guard) — no lock is taken, so calling them synchronously from inside `AtomicDo`-protected method bodies cannot deadlock.

## What Changes

- Replace all 102 `go <recv>.updateTimestamp()` call sites with direct `<recv>.updateTimestamp()` calls (pure mechanical edit; `updateTimestamp` bodies unchanged).
- Add a deterministic in-package regression test: reset `lastModifiedTimestamp` to 0, mutate, assert `GetLastModifiedTimestamp()` is updated immediately on return — this fails under the goroutine version (scheduling window) and pins the new synchronous contract.
- Delete the resolved `[2026-07-10]` updateTimestamp Follow-ups entry from `AGENTS.md`.

## Capabilities

### Modified Capabilities
- `mutation-timestamp`: `GetLastModifiedTimestamp()` now reflects a mutation immediately when the mutating call returns (previously eventually, after an unscheduled goroutine ran). Strictly stronger guarantee; timestamp granularity (seconds, monotonic non-decreasing) is unchanged.

## Impact

- **Code**: `datalist.go`, `datalist_impute.go`, `datalist_notatomic.go`, `datatable.go`, `datatable_colname.go`, `datatable_impute.go`, `datatable_name.go`, `datatable_replace.go`, `datatable_rowname.go`, `datatable_swap.go`; new test in `timestamp_sync_test.go`.
- **Docs/skills**: none — no public API or documented behavior string changes; docs never promised asynchronous timestamps.
- **Behavior**: per-mutation cost drops 2–4×; allocs/op on hot mutation paths drop by 1; no goroutine flood under tight loops. Full root/cli/stats/isr suites pass with the change (verified during the A/B run).
- **Risk**: the timestamp store now happens inside the actor-locked section instead of concurrently — it is two atomic ops, so the added lock-hold time is nanoseconds. No caller can observe ordering regressions (the old behavior was strictly less deterministic).
