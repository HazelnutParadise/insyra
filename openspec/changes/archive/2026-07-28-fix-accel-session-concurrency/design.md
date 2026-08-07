## Context
Eighteen exported methods touch session state, and several call each other: `ExecuteProjectedDataset` uses `PlanShardableWorkload` and `Report`, `updateCacheMetrics` uses `CacheSnapshot` and `Report`, `Discover` uses `Report` four times. Go's mutexes are not re-entrant, so "lock every public method" deadlocks the first time one public method calls another.

## Goals / Non-Goals
- Goals: a session any number of goroutines can share; a race detector that proves it
- Non-Goals: concurrency *within* one operation (parallel column upload, overlapping dispatches); a lock-free design; changing any public signature

## Decisions

- Decision: one mutex per session; public methods lock, unexported `*Locked` variants carry the bodies that internal callers use.
  - Rationale: the alternative — an actor goroutine per session, matching the core's `AtomicDo` — buys ordering guarantees this runtime does not need yet, at the cost of an owning goroutine per session and a channel hop per call. A mutex keeps the code shaped like the code around it. If sessions later need cross-instance atomicity the way DataLists do, the actor conversion is mechanical.
  - A plain `Mutex`, not `RWMutex`: reads are cheap clones, writes are frequent on the execution path, and the read paths all sit next to metric writes. The contention profile does not justify two lock modes until a measurement says otherwise.

- Decision: the session lock is held across backend execution, so a session serializes its own device work.
  - Rationale: releasing the lock around `executor.Execute` would let a second goroutine observe half-updated cache and report state, which is the bug this change exists to remove. The cost is that `Report()` from another goroutine waits while a kernel runs — acceptable for a runtime whose readback timeout already bounds the wait.

- Decision: serialize device submission process-wide inside `internal/wgpu`.
  - Rationale: the session lock only protects one session, but every session shares the package-level device handle, and gogpu's queue-concurrency guarantees are unverified. One mutex around `Sum`'s body costs nothing for a single device — concurrent submissions would serialize on the hardware queue anyway — and removes the dependence on an unverified upstream property.

- Decision: `ExecuteDataList` and `ExecuteDataTable` do not take the lock; `ProjectDataList` and `ProjectDataTable` lock only around cache insertion.
  - Rationale: projection is the expensive host-side step (43 ms on a 4 Mi column) and touches no session state — `projectValues` is pure and `DataList.Data()` has the core's own actor behind it. Locking around it would serialize exactly the work that can safely overlap. The state-touching tail (`cacheDataset`, `ExecuteProjectedDataset`) is what locks.

## Risks / Trade-offs
- Risk: a future public method that calls another public method deadlocks.
  - Mitigation: the pattern is mechanical — public wrapper locks, `*Locked` body does the work — and the concurrency test exercises every public method under contention, so a re-entrant call deadlocks the suite rather than a user.
- Trade-off: one session's operations no longer overlap at all.
  - Accepted: overlap never worked; it raced. Real intra-session parallelism is scheduler work for the dispatcher phase of the roadmap.
